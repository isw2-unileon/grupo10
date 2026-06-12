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

const notes = ref<Note[]>([])
const currentView = ref<'list' | 'editor' | 'upload'>('list')
const loading = ref(false)

// Estados del editor
const editNoteId = ref<string | null>(null) // Si tiene ID, estamos editando; si es null, es nuevo
const editTitle = ref('')
const editContent = ref('')

// Estados subida Word
const uploadTitle = ref('')
const selectedFile = ref<File | null>(null)

// Utilidad para las peticiones HTTP
const baseUrl = import.meta.env.VITE_API_URL || ''
function getHeaders(isFormData = false) {
  const token = localStorage.getItem('token')
  const headers: HeadersInit = {
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  }
  if (!isFormData) headers['Content-Type'] = 'application/json'
  return headers
}

// --- 1. CARGAR APUNTES (LEER) ---
async function fetchNotes() {
  loading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/notes`, { headers: getHeaders() })
    if (res.ok) notes.value = await res.json()
  } catch (e) {
    console.error("Error cargando apuntes", e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNotes()
})

// --- 2. GESTIÓN MANUAL (CREAR / EDITAR) ---
function openNewEditor() {
  editNoteId.value = null
  editTitle.value = ''
  editContent.value = ''
  currentView.value = 'editor'
}

function openEditNote(note: Note) {
  editNoteId.value = note.id
  editTitle.value = note.title
  editContent.value = note.content
  currentView.value = 'editor'
}

async function saveManualNote() {
  if (!editTitle.value || !editContent.value) return alert('Rellena el título y contenido.')
  
  const payload = { title: editTitle.value, content: editContent.value }
  const isEditing = editNoteId.value !== null
  const url = isEditing ? `${baseUrl}/api/notes/${editNoteId.value}` : `${baseUrl}/api/notes`
  const method = isEditing ? 'PUT' : 'POST'

  try {
    const res = await fetch(url, {
      method,
      headers: getHeaders(),
      body: JSON.stringify(payload)
    })
    
    if (res.ok) {
      await fetchNotes() // Recargamos la lista desde el servidor
      currentView.value = 'list'
    } else {
      alert('Error al guardar el apunte')
    }
  } catch (e) {
    console.error(e)
  }
}

// --- 3. BORRAR APUNTE ---
async function deleteNote(id: string) {
  if (!confirm('¿Seguro que quieres borrar este apunte de forma permanente?')) return
  try {
    const res = await fetch(`${baseUrl}/api/notes/${id}`, { method: 'DELETE', headers: getHeaders() })
    if (res.ok) {
      notes.value = notes.value.filter(n => n.id !== id)
    }
  } catch (e) {
    console.error(e)
  }
}

// --- 4. SUBIR DOCUMENTO WORD ---
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
    const res = await fetch(`${baseUrl}/api/notes/upload`, {
      method: 'POST',
      headers: getHeaders(true), // isFormData = true
      body: formData
    })
    
    if (res.ok) {
      await fetchNotes()
      uploadTitle.value = ''
      selectedFile.value = null
      currentView.value = 'list'
    }
  } catch (e) {
    alert('Error al subir el documento')
  }
}

// --- 5. FLUJO DE IA Y PROFESOR ---
async function requestAIReview(noteId: string) {
  try {
    // Ponemos el estado visual de carga en el apunte que corresponda (opcional)
    const res = await fetch(`${baseUrl}/api/notes/${noteId}/ai-review`, { 
      method: 'POST', 
      headers: getHeaders() 
    })
    
    // NUEVO: Si el servidor responde con un error (ej: 500), leemos el porqué
    if (!res.ok) {
      const errorTexto = await res.text()
      throw new Error(errorTexto || `Error del servidor (Código ${res.status})`)
    }
    
    // Si todo ha ido bien, recargamos los apuntes para ver el feedback de la IA
    await fetchNotes()
    alert('🤖 ¡La IA ha terminado de revisar tus apuntes!')
    
  } catch (e: any) {
    // Ahora cualquier fallo saldrá en una alerta en tu navegador
    alert('Error al solicitar revisión de IA: ' + e.message)
  }
}  

async function sendToTeacher(noteId: string) {
  if (!confirm('¿Enviar al profesor para su revisión final? Ya no podrás editarlo.')) return
  try {
    const res = await fetch(`${baseUrl}/api/notes/${noteId}/submit`, { method: 'POST', headers: getHeaders() })
    if (res.ok) await fetchNotes()
  } catch (e) {
    alert('Error al enviar al profesor')
  }
}
</script>

<template>
  <div class="notes-manager">
    <header class="manager-header">
      <h1>Mis Apuntes</h1>
      <div class="header-actions">
        <button @click="currentView = 'list'" :class="{'active': currentView === 'list'}" class="nav-btn">📋 Mis Archivos</button>
        <button @click="openNewEditor" :class="{'active': currentView === 'editor'}" class="nav-btn">✍️ Redactar</button>
        <button @click="currentView = 'upload'" :class="{'active': currentView === 'upload'}" class="nav-btn">📄 Subir Word</button>
      </div>
    </header>

    <main v-if="currentView === 'list'" class="view-list">
      <p v-if="loading">Cargando tus apuntes...</p>
      <div v-if="!loading && notes.length === 0" class="empty-state">No tienes apuntes todavía.</div>

      <div class="notes-grid">
        <article v-for="note in notes" :key="note.id" class="note-card">
          <div class="note-header">
            <h3>{{ note.title }}</h3>
            <span class="badge" :class="note.status">{{ note.status }}</span>
          </div>
          
          <p class="note-preview">{{ note.content.substring(0, 80) }}...</p>
          
          <div v-if="note.ai_feedback" class="feedback-box ai"><strong>🤖 IA:</strong> {{ note.ai_feedback }}</div>
          <div v-if="note.teacher_feedback" class="feedback-box teacher"><strong>👨‍🏫 Profe:</strong> {{ note.teacher_feedback }}</div>

          <div class="note-actions">
            <button v-if="note.status !== 'pending' && note.status !== 'approved'" @click="openEditNote(note)" class="btn-action">✏️ Editar</button>
            <button @click="deleteNote(note.id)" class="btn-action delete">🗑️ Borrar</button>
            
            <button v-if="note.status === 'draft' || note.status === 'ai_reviewed'" @click="requestAIReview(note.id)" class="btn-action ai">🤖 IA</button>
            <button v-if="note.status === 'ai_reviewed'" @click="sendToTeacher(note.id)" class="btn-action teacher">📤 Enviar</button>
          </div>
        </article>
      </div>
    </main>

    <main v-else-if="currentView === 'editor'" class="view-editor">
      <h2>{{ editNoteId ? 'Editar Apunte' : 'Nuevo Apunte' }}</h2>
      <div class="form-group">
        <label>Título:</label>
        <input v-model="editTitle" type="text" />
      </div>
      <div class="form-group">
        <label>Contenido:</label>
        <textarea v-model="editContent" rows="10"></textarea>
      </div>
      <button @click="saveManualNote" class="btn-save">💾 Guardar</button>
    </main>

    <main v-else-if="currentView === 'upload'" class="view-upload">
      <h2>Importar Documento Word</h2>
      <div class="form-group">
        <label>Título:</label>
        <input v-model="uploadTitle" type="text" />
      </div>
      <div class="form-group">
        <label>Archivo (.docx):</label>
        <input type="file" @change="handleFileChange" accept=".doc,.docx" />
      </div>
      <button @click="uploadDocument" :disabled="!selectedFile" class="btn-save">⬆️ Subir y Extraer Texto</button>
    </main>
  </div>
</template>

<style scoped>
.notes-manager { max-width: 1100px; margin: 0 auto; padding: 2rem 1rem; }
.manager-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #e2e8f0; padding-bottom: 1rem; margin-bottom: 2rem; }
.header-actions { display: flex; gap: 0.5rem; }
.nav-btn { padding: 0.5rem 1rem; border: none; background: #f1f5f9; cursor: pointer; border-radius: 0.25rem; font-weight: bold; color: #475569;}
.nav-btn.active { background: #2563eb; color: white; }

.notes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1.5rem; }
.note-card { border: 1px solid #cbd5e1; padding: 1.5rem; border-radius: 0.5rem; display: flex; flex-direction: column; background: white;}
.note-header { display: flex; justify-content: space-between; margin-bottom: 1rem;}
.note-header h3 { margin: 0; font-size: 1.1rem; }
.badge { padding: 0.2rem 0.5rem; font-size: 0.75rem; border-radius: 1rem; background: #e2e8f0; }

.feedback-box { font-size: 0.85rem; padding: 0.75rem; border-radius: 0.25rem; margin-bottom: 1rem; }
.feedback-box.ai { background: #f8fafc; border-left: 3px solid #7c3aed; }
.feedback-box.teacher { background: #f0fdf4; border-left: 3px solid #10b981; }

.note-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-top: auto; }
.btn-action { padding: 0.4rem 0.6rem; font-size: 0.8rem; cursor: pointer; border: 1px solid #cbd5e1; background: white; border-radius: 0.25rem;}
.btn-action.delete { color: #dc2626; border-color: #fca5a5; }
.btn-action.ai { background: #f3e8ff; color: #7e22ce; border-color: #d8b4fe; }
.btn-action.teacher { background: #e0f2fe; color: #0369a1; border-color: #7dd3fc; }

.form-group { margin-bottom: 1.5rem; display: flex; flex-direction: column; }
.form-group label { font-weight: bold; margin-bottom: 0.5rem; }
.form-group input, .form-group textarea { padding: 0.75rem; border: 1px solid #cbd5e1; border-radius: 0.25rem; }
.btn-save { background: #2563eb; color: white; padding: 0.75rem 1.5rem; border: none; border-radius: 0.25rem; font-weight: bold; cursor: pointer; }
</style>