<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface PendingNote {
  id: string
  title: string
  content: string
  author_email: string
  ai_feedback: string
  created_at: string
}

const pendingNotes = ref<PendingNote[]>([])
const selectedNote = ref<PendingNote | null>(null)
const teacherFeedback = ref('')
const loading = ref(false)

const baseUrl = import.meta.env.VITE_API_URL || ''
function getHeaders() {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  }
}

// 1. Cargar lista de pendientes
async function fetchPendingNotes() {
  loading.value = true
  try {
    const res = await fetch(`${baseUrl}/api/teacher/notes/pending`, { headers: getHeaders() })
    if (res.ok) pendingNotes.value = await res.json()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchPendingNotes()
})

// 2. Seleccionar un apunte para revisar
function reviewNote(note: PendingNote) {
  selectedNote.value = note
  teacherFeedback.value = ''
}

// 3. Enviar feedback y aprobar
async function approveNote() {
  if (!selectedNote.value) return
  if (!teacherFeedback.value) return alert('Por favor, escribe un feedback para el alumno.')

  try {
    const res = await fetch(`${baseUrl}/api/notes/${selectedNote.value.id}/approve`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ feedback: teacherFeedback.value })
    })

    if (res.ok) {
      alert('Apunte aprobado y devuelto al alumno.')
      selectedNote.value = null
      await fetchPendingNotes() // Refrescar la lista
    }
  } catch (e) {
    alert('Error al aprobar el apunte.')
  }
}
</script>

<template>
  <div class="teacher-dashboard">
    <header class="dashboard-header">
      <h1>Panel de Revisión</h1>
      <p>Apuntes esperando tu aprobación.</p>
    </header>

    <div class="review-layout">
      <aside class="pending-list">
        <p v-if="loading">Buscando entregas...</p>
        <p v-if="!loading && pendingNotes.length === 0" class="empty">🎉 Todo al día, no hay apuntes pendientes.</p>
        
        <div 
          v-for="note in pendingNotes" 
          :key="note.id" 
          class="pending-card"
          :class="{'active': selectedNote?.id === note.id}"
          @click="reviewNote(note)"
        >
          <h4>{{ note.title }}</h4>
          <span class="author">Por: {{ note.author_email || 'Alumno' }}</span>
        </div>
      </aside>

      <main class="review-panel">
        <div v-if="!selectedNote" class="no-selection">
          👈 Selecciona un apunte de la lista para revisarlo.
        </div>

        <div v-else class="note-viewer">
          <div class="viewer-header">
            <h2>{{ selectedNote.title }}</h2>
            <p><strong>Autor:</strong> {{ selectedNote.author_email || 'Alumno' }}</p>
          </div>

          <div class="note-content">
            <p>{{ selectedNote.content }}</p>
          </div>

          <div class="ai-reference" v-if="selectedNote.ai_feedback">
            <strong>🤖 Feedback previo de la IA:</strong>
            <p>{{ selectedNote.ai_feedback }}</p>
          </div>

          <div class="feedback-form">
            <label>Tu corrección / Feedback:</label>
            <textarea 
              v-model="teacherFeedback" 
              rows="4" 
              placeholder="Escribe aquí tus comentarios para el alumno..."
            ></textarea>
            
            <div class="actions">
              <button @click="selectedNote = null" class="btn-cancel">Cancelar</button>
              <button @click="approveNote" class="btn-approve">✅ Aprobar Apunte</button>
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.teacher-dashboard { max-width: 1200px; margin: 0 auto; padding: 2rem 1rem; }
.dashboard-header { margin-bottom: 2rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 1rem; }
.dashboard-header h1 { margin: 0; color: #1e293b; }

.review-layout { display: grid; grid-template-columns: 300px 1fr; gap: 2rem; align-items: start; }

/* Lista izquierda */
.pending-list { display: flex; flex-direction: column; gap: 1rem; }
.pending-card { background: white; border: 1px solid #cbd5e1; padding: 1rem; border-radius: 0.5rem; cursor: pointer; transition: all 0.2s; }
.pending-card:hover { border-color: #2563eb; }
.pending-card.active { border-color: #2563eb; background: #eff6ff; box-shadow: 0 0 0 2px #bfdbfe; }
.pending-card h4 { margin: 0 0 0.5rem 0; color: #0f172a; }
.author { font-size: 0.8rem; color: #64748b; }
.empty { color: #10b981; font-weight: bold; }

/* Panel derecho */
.review-panel { background: white; border: 1px solid #e2e8f0; border-radius: 0.5rem; min-height: 500px; }
.no-selection { padding: 3rem; text-align: center; color: #64748b; font-style: italic; }
.viewer-header { padding: 1.5rem; border-bottom: 1px solid #e2e8f0; background: #f8fafc; border-radius: 0.5rem 0.5rem 0 0; }
.viewer-header h2 { margin: 0 0 0.5rem 0; }
.viewer-header p { margin: 0; color: #475569; font-size: 0.9rem; }

.note-content { padding: 1.5rem; font-size: 1.05rem; line-height: 1.6; color: #334155; white-space: pre-wrap; }
.ai-reference { margin: 0 1.5rem; background: #f3e8ff; border-left: 4px solid #9333ea; padding: 1rem; border-radius: 0.25rem; font-size: 0.9rem; }

.feedback-form { padding: 1.5rem; border-top: 1px solid #e2e8f0; margin-top: 1.5rem; background: #f8fafc; border-radius: 0 0 0.5rem 0.5rem; }
.feedback-form label { display: block; font-weight: bold; margin-bottom: 0.5rem; }
.feedback-form textarea { width: 100%; padding: 0.75rem; border: 1px solid #cbd5e1; border-radius: 0.25rem; margin-bottom: 1rem; font-family: inherit; resize: vertical; box-sizing: border-box;}

.actions { display: flex; justify-content: flex-end; gap: 1rem; }
.btn-cancel { background: white; border: 1px solid #cbd5e1; padding: 0.5rem 1rem; border-radius: 0.25rem; cursor: pointer; font-weight: bold; }
.btn-approve { background: #10b981; color: white; border: none; padding: 0.5rem 1rem; border-radius: 0.25rem; cursor: pointer; font-weight: bold; }
.btn-approve:hover { background: #059669; }
</style>