<template>
  <div class="group-detail-container" style="max-width: 900px; margin: 0 auto; padding: 20px;">
    <router-link to="/teacher/groups" style="text-decoration: none; color: #008CBA; font-weight: bold;">
      ⬅️ Volver a mis grupos
    </router-link>

    <div v-if="loading" style="text-align: center; margin-top: 40px;">
      <h3>Cargando detalles de la asignatura... 🔄</h3>
    </div>

    <div v-else-if="grupo">
      <h2 style="margin-top: 20px; color: #333;">📚 {{ grupo.name }}</h2>
      <p style="color: #666; font-size: 0.9rem;">ID de la asignatura: {{ grupo.id }}</p>

      <hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;" />

      <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px;">
        
        <div>
          <div style="background: #f9f9f9; padding: 15px; border-radius: 8px; border: 1px solid #e0e0e0; margin-bottom: 20px;">
            <h3 style="margin-top: 0;">Matricular Alumno</h3>
            <div style="display: flex; gap: 10px;">
              <input 
                v-model="nuevoEmail" 
                type="email" 
                placeholder="Ej: alumno@estudiante.unileon.es"
                style="flex: 1; padding: 8px; border: 1px solid #ccc; border-radius: 4px;"
                @keyup.enter="añadirAlumno"
              />
              <button @click="añadirAlumno" style="padding: 8px 12px; background-color: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: bold;">
                ➕
              </button>
            </div>
          </div>

          <h3>👥 Alumnos Matriculados ({{ miembros.length }})</h3>
          <ul v-if="miembros.length > 0" style="list-style: none; padding: 0;">
            <li v-for="miembro in miembros" :key="miembro.id" style="padding: 10px; border-bottom: 1px solid #eee; font-size: 0.9rem;">
              📧 {{ miembro.email || miembro }}
            </li>
          </ul>
          <div v-else style="color: #777; font-style: italic; font-size: 0.9rem;">
            No hay alumnos asignados.
          </div>
        </div>

        <div>
          <div style="background: #fff8e1; padding: 15px; border-radius: 8px; border: 1px solid #ffe082; margin-bottom: 20px;">
            <h3 style="margin-top: 0; color: #b78103;">📝 Crear Nueva Tarea</h3>
            
            <div style="margin-bottom: 10px;">
              <label style="display:block; font-size: 0.85rem; font-weight: bold;">Título *</label>
              <input v-model="tareaTitulo" type="text" placeholder="Ej: Práctica 1 o Examen" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box;" />
            </div>

            <div style="margin-bottom: 10px;">
              <label style="display:block; font-size: 0.85rem; font-weight: bold;">Descripción</label>
              <textarea v-model="tareaDescripcion" placeholder="Instrucciones para los alumnos..." rows="3" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; resize: vertical;"></textarea>
            </div>

            <div style="margin-bottom: 15px;">
              <label style="display:block; font-size: 0.85rem; font-weight: bold;">Fecha de Entrega</label>
              <input v-model="tareaFecha" type="date" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box;" />
            </div>

            <button @click="crearTarea" style="width: 100%; padding: 10px; background-color: #ff9800; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: bold;">
              Asignar Tarea 🚀
            </button>
          </div>

          <h3>📅 Tareas Publicadas ({{ tareas.length }})</h3>
          <ul v-if="tareas.length > 0" style="list-style: none; padding: 0;">
            <li v-for="tarea in tareas" :key="tarea.id" style="padding: 12px; border: 1px solid #e0e0e0; border-radius: 6px; margin-bottom: 10px; background: #fff;">
              <strong style="color: #2c3e50; display: block;">{{ tarea.title }}</strong>
              <p style="font-size: 0.85rem; color: #555; margin: 5px 0;" v-if="tarea.description">{{ tarea.description }}</p>
              <span v-if="tarea.due_at" style="font-size: 0.75rem; color: #d32f2f; font-weight: bold; background: #ffebee; padding: 2px 6px; border-radius: 4px;">
                ⏰ Fin: {{ new Date(tarea.due_at).toLocaleDateString() }}
              </span>
            </li>
          </ul>
          <div v-else style="color: #777; font-style: italic; font-size: 0.9rem;">
            Aún no has creado ninguna tarea para este grupo.
          </div>
        </div>

      </div>
    </div>

    <div v-else style="text-align: center; margin-top: 40px; color: red;">
      <h3>No se ha podido cargar la información de este grupo. 🛑</h3>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const grupoId = route.params.id

// Estados Reactivos
const grupo = ref(null)
const miembros = ref([])
const tareas = ref([])
const loading = ref(true)

// Inputs Formularios
const nuevoEmail = ref('')
const tareaTitulo = ref('')
const tareaDescripcion = ref('')
const tareaFecha = ref('')

// Cargar toda la información del grupo (alumnos y tareas vienen integrados gracias a Go)
const cargarDetalleGrupo = async () => {
  try {
    loading.value = true
    const res = await fetch(`/api/groups/${grupoId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token}`
      }
    })

    if (res.ok) {
      const data = await res.json()
      grupo.value = data
      miembros.value = data.members || []
      tareas.value = data.tasks || [] // Go nos da las tareas directamente aquí
    } else {
      console.error("Error al cargar los detalles del grupo")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  } finally {
    loading.value = false
  }
}

// Matricular Alumnos
const añadirAlumno = async () => {
  const emailLimpio = nuevoEmail.value.trim()
  if (!emailLimpio) return

  try {
    const res = await fetch(`/api/groups/${grupoId}/members`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token}`
      },
      body: JSON.stringify({ emails: [emailLimpio] })
    })

    if (res.ok) {
      alert("¡Estudiante matriculado! 🎉")
      nuevoEmail.value = ''
      miembros.value = await res.json()
    } else {
      const errorData = await res.json().catch(() => ({}))
      alert("Error: " + (errorData.error || "Fallo al añadir"))
    }
  } catch (error) {
    console.error(error)
  }
}

// Crear Tarea en Go
const crearTarea = async () => {
  if (!tareaTitulo.value.trim()) {
    alert("El título de la tarea es obligatorio 🛑")
    return
  }

  // Preparamos los datos tal y como los pide la struct createTaskRequest en Go
  const payload = {
    title: tareaTitulo.value.trim(),
    description: tareaDescripcion.value.trim() || null,
    due_at: tareaFecha.value ? new Date(tareaFecha.value).toISOString() : null
  }

  try {
    const res = await fetch(`/api/groups/${grupoId}/tasks`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token}`
      },
      body: JSON.stringify(payload)
    })

    if (res.ok) {
      alert("¡Tarea asignada con éxito al grupo! 🚀")
      // Limpiamos los campos del formulario
      tareaTitulo.value = ''
      tareaDescripcion.value = ''
      tareaFecha.value = ''
      
      // Refrescamos los datos para ver la tarea reflejada abajo
      cargarDetalleGrupo()
    } else {
      const errorData = await res.json().catch(() => ({}))
      alert("Error al crear tarea: " + (errorData.error || "Fallo en el servidor"))
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  }
}

onMounted(() => {
  cargarDetalleGrupo()
})
</script>