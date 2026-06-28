<template>
  <div class="course-container" style="max-width: 900px; margin: 0 auto; padding: 20px; font-family: sans-serif;">
    <router-link to="/student/groups" style="text-decoration: none; color: #d97706; font-weight: bold;">
      ⬅️ Volver a mis asignaturas
    </router-link>

    <h2 style="margin-top: 20px; color: #1e293b; font-size: 2.2rem;">🎓 Mi Campus Virtual</h2>

    <div v-if="loading" style="text-align: center; margin-top: 40px;">
      <h3>Cargando asignaturas y materiales... 🔄</h3>
    </div>

    <div v-else>
      <div v-if="sections.length === 0" style="text-align: center; padding: 40px; background: #f1f5f9; border-radius: 12px; color: #475569;">
        <span style="font-size: 3rem;">🏝️</span>
        <h3>Aún no hay contenido disponible</h3>
        <p>Tu docente no ha colgado apuntes ni tareas en este curso.</p>
      </div>

      <div v-for="section in sections" :key="section.id" style="background: white; border: 1px solid #e2e8f0; border-radius: 12px; margin-bottom: 2rem; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); overflow: hidden;">
        <div style="background: #1e293b; color: white; padding: 15px 20px;">
          <h3 style="margin: 0; font-size: 1.25rem;">📁 {{ section.title }}</h3>
        </div>

        <div style="padding: 20px;">
          <ul v-if="section.resources && section.resources.length > 0" style="list-style: none; padding: 0; margin: 0;">
            <li v-for="resource in section.resources" :key="resource.id" style="padding: 18px; border: 1px solid #cbd5e1; border-radius: 8px; margin-bottom: 15px; background: #f8fafc;">
              
              <div style="display: flex; align-items: flex-start; gap: 15px;">
                <div style="font-size: 2.2rem;">
                  {{ resource.type === 'file' ? '📄' : resource.type === 'assignment' ? '📝' : '❓' }}
                </div>
                
                <div style="flex-grow: 1;">
                  <h4 style="margin: 0 0 5px 0; color: #0f172a; font-size: 1.2rem;">{{ resource.title }}</h4>
                  <p style="margin: 0; color: #475569; font-size: 0.95rem; white-space: pre-wrap;">{{ resource.content }}</p>
                  
                  <div v-if="resource.file_path" style="margin-top: 10px;">
                    <button @click="downloadSecureFile(resource.file_path, resource.title)" style="background: none; border: none; color: #2563eb; font-weight: bold; cursor: pointer; text-decoration: underline; padding: 0;">
                      📥 Descargar Material Adjunto
                    </button>
                  </div>

                  <div v-if="resource.due_at" style="margin-top: 10px; font-size: 0.85rem; color: #475569; font-weight: bold;">
                    ⏳ Plazo de entrega: {{ new Date(resource.due_at).toLocaleString() }}
                  </div>
                </div>

                <div style="text-align: right; min-width: 130px;">
                  <span v-if="resource.has_submitted" style="background: #dcfce3; color: #166534; padding: 4px 10px; border-radius: 20px; font-size: 0.8rem; font-weight: bold; display: block; margin-bottom: 5px; text-align: center;">
                    ✅ Entregado
                  </span>
                  <span v-if="resource.is_late" style="background: #fee2e2; color: #991b1b; padding: 4px 10px; border-radius: 20px; font-size: 0.8rem; font-weight: bold; display: block; text-align: center; margin-bottom: 5px;">
                    ⚠️ Fuera de Plazo
                  </span>
                  <span v-if="resource.current_grade !== undefined && resource.current_grade !== null" style="background: #e0f2fe; color: #0369a1; padding: 4px 10px; border-radius: 4px; font-size: 0.85rem; font-weight: bold; display: block; text-align: center;">
                    Nota: {{ resource.current_grade }}/10
                  </span>
                </div>
              </div>

              <!-- Interactive Case A: Upload Assignment -->
              <div v-if="resource.type === 'assignment' && !resource.has_submitted" style="margin-top: 15px; padding-top: 15px; border-top: 1px dashed #cbd5e1; background: #fff; padding: 15px; border-radius: 6px; border: 1px solid #e2e8f0;">
                <h5 style="margin: 0 0 10px 0; font-size: 0.95rem; color: #1e293b;">Subir mi solución</h5>
                <div style="margin-bottom: 10px;">
                  <textarea v-model="submissionTexts[resource.id]" placeholder="Escribe aquí aclaraciones para el profesor..." rows="2" style="width: 100%; padding: 8px; border: 1px solid #cbd5e1; border-radius: 4px; box-sizing: border-box; font-family: inherit;"></textarea>
                </div>
                <div style="margin-bottom: 12px;">
                  <input type="file" @change="(e) => attachStudentFile(e, resource.id)" style="font-size: 0.85rem;" />
                </div>
                <button @click="submitStudentWork(resource.id)" style="background: #d97706; color: white; padding: 8px 16px; border: none; border-radius: 4px; font-weight: bold; cursor: pointer; font-size: 0.9rem;">
                  Subir y Entregar Trabajo ⬆️
                </button>
              </div>

              <!-- Interactive Case B: Quiz Render -->
              <div v-if="resource.type === 'quiz' && !resource.has_submitted" style="margin-top: 15px; padding-top: 15px; border-top: 1px dashed #cbd5e1; background: white; padding: 15px; border-radius: 6px; text-align: center;">
                <h5 style="margin: 0 0 12px 0; color: #4f46e5; font-size: 1rem;">📝 Responder Test del Profesor</h5>
                <p style="color: #475569; font-size: 0.9rem; font-style: italic; margin-bottom: 15px;">Entraremos a la sala de test para completar las preguntas.</p>
                <button @click="$router.push(`/student/groups/${groupId}/quiz/${resource.id}`)" style="background: #4f46e5; color: white; padding: 8px 16px; border: none; border-radius: 4px; font-weight: bold; cursor: pointer;">
                  {{ resource.has_submitted ? 'Ver Mi Revisión 📊' : 'Acceder al Cuestionario ✏️' }}
                </button>
              </div>

            </li>
          </ul>
          <p v-else style="color: #94a3b8; font-style: italic; margin: 0;">No hay elementos colgados en esta sección.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const groupId = route.params.id

const sections = ref([])
const loading = ref(true)

// Dynamic student forms
const submissionTexts = ref({})
const submissionFiles = ref({})

function getHeaders(isMultipart = false) {
  const h = { 'Authorization': `Bearer ${auth.token}` }
  if (!isMultipart) h['Content-Type'] = 'application/json'
  return h
}

const loadMaterials = async () => {
  try {
    loading.value = true
    const res = await fetch(`/api/groups/${groupId}/content`, { headers: getHeaders() })
    if (res.ok) sections.value = await res.json() || []
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

const attachStudentFile = (e, resourceId) => {
  if (e.target.files && e.target.files.length > 0) {
    submissionFiles.value[resourceId] = e.target.files[0]
  }
}

const submitStudentWork = async (resourceId) => {
  const text = submissionTexts.value[resourceId] || ""
  const file = submissionFiles.value[resourceId]
  if (!file && !text) return alert("Debes adjuntar un archivo o escribir un texto para tu entrega.")

  try {
    const formData = new FormData()
    formData.append("text_content", text)
    if (file) formData.append("file", file)

    const res = await fetch(`/api/resources/${resourceId}/submit`, {
      method: 'POST',
      headers: getHeaders(true),
      body: formData
    })

    if (res.ok) {
      alert("✅ ¡Entrega enviada y guardada de forma segura en el servidor!");
      loadMaterials()
    } else {
      alert("Error al tramitar la entrega.")
    }
  } catch (e) { console.error(e) }
}

// Secure blob download (JWT authorized)
const downloadSecureFile = async (filePath, title) => {
  try {
    const res = await fetch(`/api/uploads/${filePath}`, {
      headers: { 'Authorization': `Bearer ${auth.token}` }
    })
    if (!res.ok) throw new Error("Fallo en la descarga")
    
    // Convert to blob and force download
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    // Extract extension if possible
    const ext = filePath.includes('.') ? '.' + filePath.split('.').pop() : ''
    a.download = `${title}${ext}`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    alert("No se pudo descargar el archivo.")
    console.error(error)
  }
}

onMounted(() => { loadMaterials() })
</script>