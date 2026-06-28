<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Note {
  id: string
  title: string
  content: string
  status: 'draft' | 'ai_reviewed' | 'pending' | 'approved'
  ai_feedback?: string | null
  teacher_feedback?: string | null
  created_at: string
}

// Estados de la vista: añadimos 'shared' para la comunidad
const currentView = ref<'list' | 'editor' | 'upload' | 'detail' | 'shared'>('list')
const notes = ref<Note[]>([])
const sharedNotes = ref<Note[]>([]) // Lista de apuntes de otros
const selectedNote = ref<Note | null>(null)
const viewingShared = ref(false) // Para saber si estamos leyendo un apunte propio o prestado
const loading = ref(false)
const actionLoading = ref(false)

// Estados del editor
const editNoteId = ref<string | null>(null)
const editTitle = ref('')
const editContent = ref('')

// Estados de subida
const uploadTitle = ref('')
const selectedFile = ref<File | null>(null)

// --- ESTADOS DEL MODAL DE COMPARTIR ---
const showShareModal = ref(false)
const noteToShare = ref<string | null>(null)
const shareEmail = ref('')
const shareGroupId = ref('')

const baseUrl = import.meta.env.VITE_API_URL || ''
function getHeaders(isFormData = false) {
  const token = localStorage.getItem('token')
  const headers: HeadersInit = {
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  }
  if (!isFormData) headers['Content-Type'] = 'application/json'
  return headers
}

// Cargar apuntes propios
async function fetchNotes() {
  loading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes`, { headers: getHeaders() })
    if (res.ok) {
      notes.value = await res.json()
      if (selectedNote.value && !viewingShared.value) {
        const updated = notes.value.find(n => n.id === selectedNote.value?.id)
        if (updated) selectedNote.value = updated
      }
    }
  } catch (e) {
    console.error("Error cargando apuntes", e)
  } finally {
    loading.value = false
  }
}

// Cargar apuntes que otros me han compartido
async function fetchSharedNotes() {
  loading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes/shared`, { headers: getHeaders() })
    if (res.ok) {
      sharedNotes.value = await res.json()
    }
  } catch (e) {
    console.error("Error cargando apuntes compartidos", e)
  } finally {
    loading.value = false
  }
}

// Cambiar de pestaña
function switchView(view: 'list' | 'editor' | 'upload' | 'shared') {
  currentView.value = view
  if (view === 'list') fetchNotes()
  if (view === 'shared') fetchSharedNotes()
}

onMounted(() => {
  fetchNotes()
})

// Entrar a estudiar un apunte
function viewNoteDetail(note: Note, isShared: boolean = false) {
  selectedNote.value = note
  viewingShared.value = isShared
  currentView.value = 'detail'
}

function openNewEditor() {
  editNoteId.value = null
  editTitle.value = ''
  editContent.value = ''
  currentView.value = 'editor'
}

function openEditFromDetail() {
  if (!selectedNote.value) return
  editNoteId.value = selectedNote.value.id
  editTitle.value = selectedNote.value.title
  editContent.value = selectedNote.value.content
  currentView.value = 'editor'
}

async function saveManualNote() {
  if (!editTitle.value || !editContent.value) return alert('Rellena el título y contenido.')
  const payload = { title: editTitle.value, content: editContent.value }
  const isEditing = editNoteId.value !== null
  const url = isEditing ? `${baseUrl}/api/notes/${editNoteId.value}` : `${baseUrl}/api/notes`
  const method = isEditing ? 'PUT' : 'POST'

  try {
    const res = await fetch(url, { method, headers: getHeaders(), body: JSON.stringify(payload) })
    if (res.ok) {
      await fetchNotes()
      currentView.value = isEditing && selectedNote.value ? 'detail' : 'list'
    } else {
      alert('Error al guardar')
    }
  } catch (e) { console.error(e) }
}

async function deleteNote(id: string, event?: Event) {
  if (event) event.stopPropagation()
  if (!confirm('¿Seguro que quieres borrar este apunte?')) return
  try {
    const res = await fetch(`${baseUrl}/api/notes/${id}`, { method: 'DELETE', headers: getHeaders() })
    if (res.ok) {
      notes.value = notes.value.filter(n => n.id !== id)
      if (currentView.value === 'detail') currentView.value = 'list'
    }
  } catch (e) { console.error(e) }
}

async function uploadDocument() {
  if (!uploadTitle.value || !selectedFile.value) return alert('Faltan datos.')
  const formData = new FormData()
  formData.append('file', selectedFile.value)
  formData.append('title', uploadTitle.value)

  try {
    const res = await fetch(`${baseUrl}/api/notes/upload`, { method: 'POST', headers: getHeaders(true), body: formData })
    if (res.ok) {
      await fetchNotes()
      uploadTitle.value = ''
      selectedFile.value = null
      currentView.value = 'list'
    } else {
      alert(`No se pudo importar el documento: ${(await res.text()) || res.statusText}`)
    }
  } catch (e) { alert('Error al subir documento') }
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) selectedFile.value = input.files[0]
}

// --- ACCIONES DE IA Y PROFESOR ---
async function requestAIReview(noteId: string) {
  actionLoading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes/${noteId}/ai-review`, { method: 'POST', headers: getHeaders() })
    if (!res.ok) { const errMsg = await res.text(); throw new Error(errMsg) }
    await fetchNotes()
    alert('🤖 ¡La IA ha terminado de revisar tus apuntes!')
  } catch (e: any) { alert('Error con la IA: ' + e.message) }
  finally { actionLoading.value = false }
}

async function sendToTeacher(noteId: string) {
  if (!confirm('¿Enviar al profesor? Ya no podrás editarlo manualmente.')) return
  actionLoading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes/${noteId}/submit`, { method: 'POST', headers: getHeaders() })
    if (res.ok) { await fetchNotes(); alert('🚀 Enviado al profesor.') }
  } catch (e) { alert('Error al enviar') }
  finally { actionLoading.value = false }
}

// --- LÓGICA DE COMPARTIR ---
function openShareDialog(noteId: string) {
  noteToShare.value = noteId
  shareEmail.value = ''
  shareGroupId.value = ''
  showShareModal.value = true
}

async function submitShare() {
  if (!shareEmail.value && !shareGroupId.value) {
    return alert('Debes introducir un email o un ID de grupo para compartir.')
  }
  
  actionLoading.value = true
  try {
    const payload = {
      email: shareEmail.value || undefined,
      group_id: shareGroupId.value || undefined
    }
    
    const res = await fetch(`${baseUrl}/api/notes/${noteToShare.value}/share`, { 
      method: 'POST', 
      headers: getHeaders(),
      body: JSON.stringify(payload)
    })
    
    if (res.ok) {
      alert('✅ ¡Apunte compartido con éxito!')
      showShareModal.value = false
    } else {
      const text = await res.text()
      alert('Error al compartir: ' + text)
    }
  } catch (e) {
    alert('Error de conexión al compartir.')
  } finally {
    actionLoading.value = false
  }
}

function formatStatus(status: string) {
  switch (status) {
    case 'draft': return 'Borrador'
    case 'ai_reviewed': return 'Revisado por IA'
    case 'pending': return 'En Revisión (Profesor)'
    case 'approved': return 'Aprobado'
    default: return status
  }
}
</script>

<template>
  <div class="notes-manager">
    <div v-if="showShareModal" class="modal-overlay" @click.self="showShareModal = false">
      <div class="modal-content">
        <h3>🔗 Compartir Apunte</h3>
        <p class="modal-desc">Escribe el email de un compañero o el ID de un grupo de clase para darle acceso de lectura a este documento.</p>
        
        <div class="form-group">
          <label>Email del compañero:</label>
          <input v-model="shareEmail" type="email" placeholder="ejemplo@estudiante.com" />
        </div>
        
        <div class="divider-text"><span>O TAMBIÉN</span></div>
        
        <div class="form-group">
          <label>ID del Grupo (Clase):</label>
          <input v-model="shareGroupId" type="text" placeholder="Ej: 550e8400-e29b-41d4-a716-446655440000" />
        </div>
        
        <div class="modal-actions">
          <button @click="showShareModal = false" class="btn-cancel">Cancelar</button>
          <button @click="submitShare" :disabled="actionLoading" class="btn-save">
            {{ actionLoading ? 'Compartiendo...' : 'Compartir Ahora' }}
          </button>
        </div>
      </div>
    </div>

    <header class="manager-header" v-if="currentView !== 'detail'">
      <div class="title-area">
        <h1>Mis Apuntes</h1>
        <p class="subtitle">Gestiona, estudia y comparte tus notas con la comunidad</p>
      </div>
      <div class="header-actions">
        <button @click="switchView('list')" :class="{'active': currentView === 'list'}" class="nav-btn">📋 Biblioteca</button>
        <button @click="switchView('shared')" :class="{'active': currentView === 'shared'}" class="nav-btn shared-tab">🤝 Comunidad</button>
        <button @click="switchView('editor')" :class="{'active': currentView === 'editor'}" class="nav-btn">✍️ Redactar</button>
        <button @click="switchView('upload')" :class="{'active': currentView === 'upload'}" class="nav-btn">📄 Importar</button>
      </div>
    </header>

    <main v-if="currentView === 'list'" class="view-list">
      <p v-if="loading" class="info-msg">Cargando tus carpetas...</p>
      <div v-if="!loading && notes.length === 0" class="empty-state">
        <div class="empty-icon">📚</div>
        <h3>Tu biblioteca está vacía</h3>
        <p>Crea un apunte manual o importa un archivo (.docx) para empezar.</p>
      </div>

      <div class="notes-grid" v-if="notes.length > 0">
        <article v-for="note in notes" :key="note.id" class="bubble-card" @click="viewNoteDetail(note, false)">
          <div class="card-top">
            <span class="badge" :class="note.status">{{ formatStatus(note.status) }}</span>
          </div>
          <h3 class="card-title">{{ note.title }}</h3>
          <p class="card-preview">{{ note.content.substring(0, 110) }}...</p>
          <div class="card-footer">
            <button class="btn-card-open">👁️ Ver y Estudiar</button>
            <button @click="deleteNote(note.id, $event)" class="btn-card-delete" title="Borrar apunte">🗑️</button>
          </div>
        </article>
      </div>
    </main>

    <main v-if="currentView === 'shared'" class="view-list">
      <p v-if="loading" class="info-msg">Buscando apuntes de la comunidad...</p>
      <div v-if="!loading && sharedNotes.length === 0" class="empty-state">
        <div class="empty-icon">🌱</div>
        <h3>Nadie ha compartido nada contigo aún</h3>
        <p>Pídele a tus compañeros que te compartan sus apuntes por email.</p>
      </div>

      <div class="notes-grid" v-if="sharedNotes.length > 0">
        <article v-for="note in sharedNotes" :key="note.id" class="bubble-card shared-card" @click="viewNoteDetail(note, true)">
          <div class="card-top">
            <span class="badge shared-badge">🤝 Prestado</span>
          </div>
          <h3 class="card-title">{{ note.title }}</h3>
          <p class="card-preview">{{ note.content.substring(0, 110) }}...</p>
          <div class="card-footer">
            <button class="btn-card-open">📖 Leer Apunte</button>
          </div>
        </article>
      </div>
    </main>

    <main v-else-if="currentView === 'detail' && selectedNote" class="view-detail">
      <div class="detail-navigation">
        <button @click="viewingShared ? switchView('shared') : switchView('list')" class="btn-back" style="color: black;">⬅️ Volver atrás</button>
        <span v-if="!viewingShared" class="badge status-indicator" :class="selectedNote.status">{{ formatStatus(selectedNote.status) }}</span>
        <span v-else class="badge shared-badge status-indicator">🤝 Apunte de la Comunidad</span>
      </div>
      
      <div class="study-dashboard">
        <section class="notebook-sheet">
          <div class="notebook-header-actions">
            <h2 class="notebook-title">{{ selectedNote.title }}</h2>
            
            <div class="notebook-top-buttons" v-if="!viewingShared">
              <button @click="openShareDialog(selectedNote.id)" class="btn-note-share">🔗 Compartir</button>

              <button v-if="selectedNote.status !== 'pending' && selectedNote.status !== 'approved'" @click="openEditFromDetail" class="btn-note-edit" style="color: black;">✏️ Modificar</button>
              
              <button @click="requestAIReview(selectedNote.id)" :disabled="actionLoading" class="btn-note-ai">
                {{ actionLoading ? '🤖 Pensando...' : '🤖 Consultar IA' }}
              </button>
              
              <button v-if="selectedNote.status !== 'pending' && selectedNote.status !== 'approved'" @click="sendToTeacher(selectedNote.id)" :disabled="actionLoading" class="btn-note-teacher">
                📤 Entregar
              </button>
            </div>
          </div>
          
          <hr class="notebook-divider" />
          <div class="notebook-body">{{ selectedNote.content }}</div>
        </section>

        <aside class="study-sidebar" v-if="selectedNote.ai_feedback || selectedNote.teacher_feedback">
          <div v-if="selectedNote.ai_feedback" class="feedback-bubble ai">
            <div class="bubble-title">🤖 Informe IA</div>
            <p class="bubble-text">{{ selectedNote.ai_feedback }}</p>
          </div>
          <div v-if="selectedNote.teacher_feedback" class="feedback-bubble teacher">
            <div class="bubble-title">👨‍🏫 Corrección Oficial</div>
            <p class="bubble-text">{{ selectedNote.teacher_feedback }}</p>
          </div>
        </aside>
      </div>
    </main>

    <main v-else-if="currentView === 'editor'" class="view-editor">
      <div class="editor-box">
        <h2>{{ editNoteId ? '✍️ Modificar Apunte' : '✍️ Crear Nuevo Apunte' }}</h2>
        <div class="form-group">
          <label>Título</label>
          <input v-model="editTitle" type="text" placeholder="Ej: Tema 4..." />
        </div>
        <div class="form-group">
          <label>Cuerpo</label>
          <textarea v-model="editContent" rows="15" placeholder="Escribe aquí..."></textarea>
        </div>
        <div class="form-actions">
          <button @click="editNoteId ? currentView = 'detail' : currentView = 'list'" class="btn-cancel" style="color: black;">Cancelar</button>
          <button @click="saveManualNote" class="btn-save">💾 Guardar</button>
        </div>
      </div>
    </main>

    <main v-else-if="currentView === 'upload'" class="view-upload">
      <div class="upload-container">
        <h2>📄 Importar documento</h2>
        <div class="upload-card">
          <div class="form-group">
            <label>Título</label>
            <input v-model="uploadTitle" type="text" placeholder="Ej: Apuntes SO" />
          </div>
          <div class="form-group file-wrapper">
            <label class="custom-file-upload">
              <input type="file" @change="handleFileChange" accept=".docx" />
              📁 Elegir (.docx)
            </label>
            <span class="file-name-preview" v-if="selectedFile">{{ selectedFile.name }}</span>
          </div>
          <button @click="uploadDocument" :disabled="!selectedFile" class="btn-upload-submit">Subir ⬆️</button>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* ESTILOS BASE */
.notes-manager { max-width: 1200px; margin: 0 auto; padding: 2.5rem 1rem; font-family: 'Segoe UI', sans-serif; color: #1e293b; }
.manager-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #e2e8f0; padding-bottom: 1.5rem; margin-bottom: 2.5rem; }
.title-area h1 { margin: 0 0 0.25rem 0; font-size: 2.25rem; color: #0f172a; font-weight: 800; }
.subtitle { margin: 0; color: #64748b; font-size: 1rem; }
.header-actions { display: flex; gap: 0.5rem; background: #f1f5f9; padding: 0.4rem; border-radius: 0.75rem; flex-wrap: wrap; }
.nav-btn { padding: 0.6rem 1.2rem; border: none; background: transparent; cursor: pointer; border-radius: 0.5rem; font-weight: 700; color: #64748b; transition: all 0.2s ease; }
.nav-btn.active { background: white; color: #2563eb; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }
.nav-btn.shared-tab.active { color: #059669; }

/* MODAL DE COMPARTIR */
.modal-overlay { position: fixed; top: 0; left: 0; width: 100vw; height: 100vh; background: rgba(15, 23, 42, 0.6); display: flex; justify-content: center; align-items: center; z-index: 1000; backdrop-filter: blur(4px); }
.modal-content { background: white; padding: 2.5rem; border-radius: 1.25rem; width: 90%; max-width: 450px; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.1); }
.modal-content h3 { margin: 0 0 0.5rem 0; font-size: 1.5rem; color: #0f172a; }
.modal-desc { color: #64748b; font-size: 0.95rem; margin-bottom: 1.5rem; line-height: 1.5; }
.divider-text { text-align: center; margin: 1rem 0; position: relative; }
.divider-text::before { content: ''; position: absolute; left: 0; top: 50%; width: 100%; height: 1px; background: #e2e8f0; z-index: 1; }
.divider-text span { background: white; padding: 0 10px; color: #94a3b8; font-size: 0.8rem; font-weight: 700; position: relative; z-index: 2; }
.modal-actions { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 2rem; }

/* TARJETAS */
.notes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1.75rem; }
.bubble-card { background: white; border: 1px solid #e2e8f0; border-radius: 1rem; padding: 1.5rem; display: flex; flex-direction: column; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.03); cursor: pointer; transition: all 0.2s ease; }
.bubble-card:hover { transform: translateY(-4px); box-shadow: 0 10px 15px -3px rgba(37,99,235,0.1); border-color: #bfdbfe; }
.shared-card:hover { box-shadow: 0 10px 15px -3px rgba(5,150,105,0.1); border-color: #a7f3d0; }
.card-top { margin-bottom: 1rem; }
.card-title { margin: 0 0 0.75rem 0; font-size: 1.25rem; font-weight: 700; color: #0f172a; }
.card-preview { color: #475569; font-size: 0.95rem; line-height: 1.6; flex-grow: 1; margin-bottom: 1.5rem; }
.card-footer { display: flex; justify-content: space-between; align-items: center; border-top: 1px solid #f1f5f9; padding-top: 1rem; }
.btn-card-open { background: #eff6ff; color: #2563eb; border: none; padding: 0.5rem 1rem; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }
.btn-card-delete { background: transparent; border: none; font-size: 1.1rem; cursor: pointer; }

/* ETIQUETAS */
.badge { padding: 0.35rem 0.8rem; font-size: 0.75rem; border-radius: 9999px; font-weight: 700; display: inline-block; text-transform: uppercase; }
.badge.draft { background: #f1f5f9; color: #475569; }
.badge.ai_reviewed { background: #f3e8ff; color: #7e22ce; }
.badge.pending { background: #fef3c7; color: #b45309; }
.badge.approved { background: #dcfce3; color: #166534; }
.shared-badge { background: #dcfce3; color: #059669; border: 1px solid #6ee7b7; }

/* ZONA DE ESTUDIO */
.detail-navigation { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
.btn-back { background: white; border: 1px solid #cbd5e1; padding: 0.6rem 1.2rem; border-radius: 0.5rem; cursor: pointer; font-weight: 700; }
.study-dashboard { display: grid; grid-template-columns: 2fr 1fr; gap: 2.5rem; align-items: start; }
@media (max-width: 900px) { .study-dashboard { grid-template-columns: 1fr; } }
.notebook-sheet { background: #ffffff; border: 1px solid #e2e8f0; border-radius: 1.25rem; padding: 2.5rem; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.02); }
.notebook-header-actions { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; }
.notebook-title { margin: 0; font-size: 1.8rem; color: #0f172a; font-weight: 800; flex: 1; }
.notebook-top-buttons { display: flex; gap: 0.5rem; flex-wrap: wrap; }

.btn-note-share { padding: 0.5rem 1rem; background: #eff6ff; color: #2563eb; border: 1px solid #bfdbfe; border-radius: 0.375rem; font-weight: 700; cursor: pointer; }
.btn-note-share:hover { background: #dbeafe; }
.btn-note-edit { padding: 0.5rem 1rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.375rem; font-weight: 700; cursor: pointer; }
.btn-note-ai { padding: 0.5rem 1rem; background: #7c3aed; color: white; border: none; border-radius: 0.375rem; font-weight: 700; cursor: pointer; }
.btn-note-teacher { padding: 0.5rem 1rem; background: #10b981; color: white; border: none; border-radius: 0.375rem; font-weight: 700; cursor: pointer; }

.notebook-divider { border: 0; height: 2px; background: #f1f5f9; margin: 1.5rem 0; }
.notebook-body { font-size: 1.15rem; line-height: 1.8; color: #2d3748; white-space: pre-wrap; text-align: justify; }

/* FEEDBACK & FORMULARIOS */
.study-sidebar { display: flex; flex-direction: column; gap: 1.5rem; }
.feedback-bubble { padding: 1.5rem; border-radius: 1rem; border-left: 5px solid; }
.bubble-title { font-weight: 700; text-transform: uppercase; margin-bottom: 0.75rem; }
.feedback-bubble.ai { background: #faf5ff; border-color: #a855f7; color: #5b21b6; }
.feedback-bubble.teacher { background: #f0fdf4; border-color: #22c55e; color: #14532d; }

.form-group { margin-bottom: 1.5rem; display: flex; flex-direction: column; }
.form-group label { font-weight: 700; margin-bottom: 0.5rem; color: #334155; }
.form-group input, .form-group textarea { padding: 0.85rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; font-family: inherit; }
.form-actions { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
.btn-cancel { padding: 0.75rem 1.5rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }
.btn-save { background: #2563eb; color: white; padding: 0.75rem 1.5rem; border: none; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }

.editor-box, .upload-container { background: white; border: 1px solid #e2e8f0; padding: 2.5rem; border-radius: 1.25rem; max-width: 800px; margin: 0 auto; }
.upload-card { background: #f8fafc; border: 2px dashed #cbd5e1; padding: 2rem; border-radius: 1rem; margin-top: 1.5rem; }
.custom-file-upload { display: inline-block; padding: 0.75rem 1.2rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }
.custom-file-upload input[type="file"] { display: none; }
.btn-upload-submit { background: #2563eb; color: white; padding: 0.75rem 1.5rem; border: none; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }

.empty-state { text-align: center; color: #64748b; padding: 4rem 2rem; background: white; border: 2px dashed #e2e8f0; border-radius: 1.25rem; max-width: 500px; margin: 2rem auto; }
.empty-icon { font-size: 3.5rem; margin-bottom: 1rem; }
.empty-state h3 { margin: 0 0 0.5rem 0; color: #0f172a; }
</style>