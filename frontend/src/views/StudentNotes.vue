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

const currentView = ref<'list' | 'editor' | 'upload' | 'detail'>('list')
const notes = ref<Note[]>([])
const selectedNote = ref<Note | null>(null)
const loading = ref(false)
const actionLoading = ref(false)

// Estados del editor manual
const editNoteId = ref<string | null>(null)
const editTitle = ref('')
const editContent = ref('')

// Estados de subida de Word
const uploadTitle = ref('')
const selectedFile = ref<File | null>(null)

const baseUrl = import.meta.env.VITE_API_URL || ''
function getHeaders(isFormData = false) {
  const token = localStorage.getItem('token')
  const headers: HeadersInit = {
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  }
  if (!isFormData) headers['Content-Type'] = 'application/json'
  return headers
}

async function fetchNotes() {
  loading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes`, { headers: getHeaders() })
    if (res.ok) {
      notes.value = await res.json()
      if (selectedNote.value) {
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

onMounted(() => {
  fetchNotes()
})

function viewNoteDetail(note: Note) {
  selectedNote.value = note
  currentView.value = 'detail'
}

// Abrir editor para un apunte nuevo
function openNewEditor() {
  editNoteId.value = null
  editTitle.value = ''
  editContent.value = ''
  currentView.value = 'editor'
}

// Abrir editor desde la vista de detalle
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

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) selectedFile.value = input.files[0]
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
    }
  } catch (e) { alert('Error al subir documento') }
}

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
    <header class="manager-header" v-if="currentView !== 'detail'">
      <div class="title-area">
        <h1>Mis Apuntes</h1>
        <p class="subtitle">Gestiona, estudia y optimiza tus notas con Inteligencia Artificial</p>
      </div>
      <div class="header-actions">
        <button @click="currentView = 'list'" :class="{'active': currentView === 'list'}" class="nav-btn">📋 Mi Biblioteca</button>
        <button @click="openNewEditor" :class="{'active': currentView === 'editor'}" class="nav-btn">✍️ Redactar Nota</button>
        <button @click="currentView = 'upload'" :class="{'active': currentView === 'upload'}" class="nav-btn">📄 Subir Word</button>
      </div>
    </header>

    <main v-if="currentView === 'list'" class="view-list">
      <p v-if="loading" class="info-msg">Cargando tus carpetas...</p>
      <div v-if="!loading && notes.length === 0" class="empty-state">
        <div class="empty-icon">📚</div>
        <h3>Tu biblioteca está vacía</h3>
        <p>Crea un apunte manual o sube un archivo de Word para empezar.</p>
      </div>

      <div class="notes-grid" v-if="notes.length > 0">
        <article v-for="note in notes" :key="note.id" class="bubble-card" @click="viewNoteDetail(note)">
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

    <main v-else-if="currentView === 'detail' && selectedNote" class="view-detail">
      <div class="detail-navigation">
        <button @click="currentView = 'list'" class="btn-back">⬅️</button>
        <span class="badge status-indicator" :class="selectedNote.status">{{ formatStatus(selectedNote.status) }}</span>
      </div>
      
      <div class="study-dashboard">
        <section class="notebook-sheet">
          <div class="notebook-header-actions">
            <h2 class="notebook-title">{{ selectedNote.title }}</h2>
            
            <div class="notebook-top-buttons">
              <button 
                v-if="selectedNote.status !== 'pending' && selectedNote.status !== 'approved'"
                @click="openEditFromDetail" 
                class="btn-note-edit"
              >
                ✏️ Modificar
              </button>
              
              <button 
                @click="requestAIReview(selectedNote.id)" 
                :disabled="actionLoading" 
                class="btn-note-ai"
              >
                {{ actionLoading ? '🤖 Procesando...' : '🤖 Consultar IA' }}
              </button>
              
              <button 
                v-if="selectedNote.status !== 'pending' && selectedNote.status !== 'approved'"
                @click="sendToTeacher(selectedNote.id)" 
                :disabled="actionLoading" 
                class="btn-note-teacher"
              >
                📤 Solicitar revisión
              </button>
            </div>
          </div>
          
          <hr class="notebook-divider" />
          <div class="notebook-body">{{ selectedNote.content }}</div>
        </section>

        <aside class="study-sidebar" v-if="selectedNote.ai_feedback || selectedNote.teacher_feedback">
          <div v-if="selectedNote.ai_feedback" class="feedback-bubble ai">
            <div class="bubble-title">🤖 Informe de la Inteligencia Artificial</div>
            <p class="bubble-text">{{ selectedNote.ai_feedback }}</p>
          </div>
          
          <div v-if="selectedNote.teacher_feedback" class="feedback-bubble teacher">
            <div class="bubble-title">👨‍🏫 Corrección Oficial del Docente</div>
            <p class="bubble-text">{{ selectedNote.teacher_feedback }}</p>
          </div>
        </aside>
      </div>
    </main>

    <main v-else-if="currentView === 'editor'" class="view-editor">
      <div class="editor-box">
        <h2>{{ editNoteId ? '✍️ Modificar Apunte' : '✍️ Crear Nuevo Apunte' }}</h2>
        <div class="form-group">
          <label>Título del Apunte</label>
          <input v-model="editTitle" type="text" placeholder="Ej: Tema 4 - Arquitecturas de Red" />
        </div>
        <div class="form-group">
          <label>Cuerpo del Documento</label>
          <textarea v-model="editContent" rows="15" placeholder="Escribe o pega el contenido temático completo para poder estudiar más tarde..."></textarea>
        </div>
        <div class="form-actions">
          <button @click="editNoteId ? currentView = 'detail' : currentView = 'list'" class="btn-cancel">Cancelar</button>
          <button @click="saveManualNote" class="btn-save">💾 Guardar en Biblioteca</button>
        </div>
      </div>
    </main>

    <main v-else-if="currentView === 'upload'" class="view-upload">
      <div class="upload-container">
        <h2>📄 Importar desde archivo Microsoft Word</h2>
        <p class="upload-help">Subiendo un archivo .docx, el sistema leerá los párrafos automáticamente y los transformará en apuntes interactivos estructurados.</p>
        <div class="upload-card">
          <div class="form-group">
            <label>Asigna un título</label>
            <input v-model="uploadTitle" type="text" placeholder="Ej: Apuntes de Sistemas Operativos" />
          </div>
          <div class="form-group file-wrapper">
            <label class="custom-file-upload">
              <input type="file" @change="handleFileChange" accept=".docx" />
              📁 Seleccionar Archivo (.docx)
            </label>
            <span class="file-name-preview" v-if="selectedFile">Archivo: <strong>{{ selectedFile.name }}</strong></span>
          </div>
          <button @click="uploadDocument" :disabled="!selectedFile" class="btn-upload-submit">Subir y Descomprimir Texto ⬆️</button>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.notes-manager { max-width: 1200px; margin: 0 auto; padding: 2.5rem 1rem; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; color: #1e293b; }

/* CABECERA PRINCIPAL */
.manager-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #e2e8f0; padding-bottom: 1.5rem; margin-bottom: 2.5rem; }
.title-area h1 { margin: 0 0 0.25rem 0; font-size: 2.25rem; color: #0f172a; font-weight: 800; }
.subtitle { margin: 0; color: #64748b; font-size: 1rem; }
.header-actions { display: flex; gap: 0.5rem; background: #f1f5f9; padding: 0.4rem; border-radius: 0.75rem; }
.nav-btn { padding: 0.6rem 1.2rem; border: none; background: transparent; cursor: pointer; border-radius: 0.5rem; font-weight: 700; color: #64748b; transition: all 0.2s ease; }
.nav-btn.active { background: white; color: #2563eb; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }

/* SISTEMA DE REJILLA / TARJETAS */
.notes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1.75rem; }
.bubble-card { background: white; border: 1px solid #e2e8f0; border-radius: 1rem; padding: 1.5rem; display: flex; flex-direction: column; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.03); cursor: pointer; transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1); }
.bubble-card:hover { transform: translateY(-4px); box-shadow: 0 10px 15px -3px rgba(37,99,235,0.1); border-color: #bfdbfe; }
.card-top { margin-bottom: 1rem; }
.card-title { margin: 0 0 0.75rem 0; font-size: 1.25rem; font-weight: 700; color: #0f172a; line-height: 1.4; }
.card-preview { color: #475569; font-size: 0.95rem; line-height: 1.6; margin: 0 0 1.5rem 0; flex-grow: 1; }
.card-footer { display: flex; justify-content: space-between; align-items: center; margin-top: auto; border-top: 1px solid #f1f5f9; padding-top: 1rem; }
.btn-card-open { background: #eff6ff; color: #2563eb; border: none; padding: 0.5rem 1rem; border-radius: 0.5rem; font-weight: 700; font-size: 0.85rem; cursor: pointer; }
.btn-card-open:hover { background: #dbeafe; }
.btn-card-delete { background: transparent; border: none; font-size: 1.1rem; cursor: pointer; padding: 0.4rem; border-radius: 0.375rem; }
.btn-card-delete:hover { background: #fef2f2; color: #dc2626; }

/* BADGES ESTILIZADAS */
.badge { padding: 0.35rem 0.8rem; font-size: 0.75rem; border-radius: 9999px; font-weight: 700; display: inline-block; text-transform: uppercase; letter-spacing: 0.05em; }
.badge.draft { background: #f1f5f9; color: #475569; }
.badge.ai_reviewed { background: #f3e8ff; color: #7e22ce; }
.badge.pending { background: #fef3c7; color: #b45309; }
.badge.approved { background: #dcfce3; color: #166534; }
.status-indicator { margin-bottom: 0; }

/* PANEL DE ESTUDIO (DETALLE) */
.detail-navigation { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
.btn-back { background: white; border: 1px solid #cbd5e1; padding: 0.6rem 1.2rem; border-radius: 0.5rem; cursor: pointer; font-weight: 700; font-size: 0.9rem; }
.btn-back:hover { background: #f8fafc; }
.study-dashboard { display: grid; grid-template-columns: 2fr 1fr; gap: 2.5rem; align-items: start; }
@media (max-width: 900px) { .study-dashboard { grid-template-columns: 1fr; } }

/* El folio del cuaderno */
.notebook-sheet { background: #ffffff; border: 1px solid #e2e8f0; border-radius: 1.25rem; padding: 2.5rem; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.02); }
.notebook-header-actions { display: flex; justify-content: space-between; align-items: center; gap: 1.5rem; flex-wrap: wrap; margin-bottom: 0.5rem; }
.notebook-title { margin: 0; font-size: 1.8rem; color: #0f172a; font-weight: 800; flex: 1; }
.notebook-top-buttons { display: flex; gap: 0.5rem; }

.notebook-divider { border: 0; height: 2px; background: #f1f5f9; margin-bottom: 1.75rem; margin-top: 1rem; }
.notebook-body { font-size: 1.15rem; line-height: 1.8; color: #2d3748; white-space: pre-wrap; margin-bottom: 1rem; text-align: justify; }

/* Botones reubicados arriba (Estilos limpios) */
.btn-note-edit { padding: 0.5rem 1rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.375rem; font-weight: 700; font-size: 0.85rem; cursor: pointer; color: #475569; }
.btn-note-edit:hover { background: #f8fafc; }
.btn-note-ai { padding: 0.5rem 1rem; background: #7c3aed; color: white; border: none; border-radius: 0.375rem; font-weight: 700; font-size: 0.85rem; cursor: pointer; }
.btn-note-ai:hover:not(:disabled) { background: #6d28d9; }
.btn-note-teacher { padding: 0.5rem 1rem; background: #10b981; color: white; border: none; border-radius: 0.375rem; font-weight: 700; font-size: 0.85rem; cursor: pointer; }
.btn-note-teacher:hover:not(:disabled) { background: #059669; }
button:disabled { opacity: 0.5; cursor: not-allowed; }

/* Sidebar de feedback */
.study-sidebar { display: flex; flex-direction: column; gap: 1.5rem; }
.feedback-bubble { padding: 1.5rem; border-radius: 1rem; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.02); border-left: 5px solid; }
.bubble-title { font-weight: 700; font-size: 0.95rem; text-transform: uppercase; letter-spacing: 0.02em; margin-bottom: 0.75rem; }
.bubble-text { margin: 0; font-size: 1rem; line-height: 1.6; }
.feedback-bubble.ai { background: #faf5ff; border-color: #a855f7; color: #5b21b6; }
.feedback-bubble.teacher { background: #f0fdf4; border-color: #22c55e; color: #14532d; }

/* FORMULARIOS */
.editor-box, .upload-container { background: white; border: 1px solid #e2e8f0; padding: 2.5rem; border-radius: 1.25rem; max-width: 800px; margin: 0 auto; }
.form-group { margin-bottom: 1.5rem; display: flex; flex-direction: column; }
.form-group label { font-weight: 700; margin-bottom: 0.5rem; color: #334155; font-size: 0.95rem; }
.form-group input, .form-group textarea { padding: 0.85rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; font-family: inherit; font-size: 1rem; }
.form-group input:focus, .form-group textarea:focus { outline: none; border-color: #2563eb; }

.form-actions { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
.btn-cancel { padding: 0.75rem 1.5rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.5rem; cursor: pointer; font-weight: 700; }
.btn-save, .btn-upload-submit { background: #2563eb; color: white; padding: 0.75rem 1.5rem; border: none; border-radius: 0.5rem; font-weight: 700; cursor: pointer; }
.btn-save:hover, .btn-upload-submit:hover { background: #1d4ed8; }

.upload-card { background: #f8fafc; border: 2px dashed #cbd5e1; padding: 2rem; border-radius: 1rem; margin-top: 1.5rem; }
.upload-help { color: #64748b; font-size: 1rem; line-height: 1.5; margin: 0; }
.custom-file-upload { display: inline-block; padding: 0.75rem 1.2rem; background: white; border: 1px solid #cbd5e1; border-radius: 0.5rem; font-weight: 700; cursor: pointer; margin-bottom: 0.5rem; }
.custom-file-upload input[type="file"] { display: none; }
.file-name-preview { display: block; font-size: 0.9rem; color: #059669; margin-top: 0.25rem; }

.empty-state { text-align: center; color: #64748b; padding: 4rem 2rem; background: white; border: 2px dashed #e2e8f0; border-radius: 1.25rem; max-width: 500px; margin: 2rem auto; }
.empty-icon { font-size: 3.5rem; margin-bottom: 1rem; }
.empty-state h3 { margin: 0 0 0.5rem 0; color: #0f172a; font-size: 1.35rem; }
.empty-state p { margin: 0; font-size: 0.95rem; }
</style>