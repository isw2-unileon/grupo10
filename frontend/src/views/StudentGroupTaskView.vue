<template>
  <div class="tasks-container" style="max-width: 800px; margin: 0 auto; padding: 20px;">
    <router-link to="/student/groups" style="text-decoration: none; color: #ff9800; font-weight: bold;">
      ⬅️ Volver a mis asignaturas
    </router-link>

    <h2 style="margin-top: 20px; color: #333;">📝 Tareas de la Asignatura</h2>

    <div v-if="loading" style="text-align: center; margin-top: 40px;">
      <h3>Cargando tus tareas... 🔄</h3>
    </div>

    <div v-else>
      <ul v-if="tareas.length > 0" style="list-style: none; padding: 0;">
        <li 
          v-for="tarea in tareas" 
          :key="tarea.id"
          style="border: 1px solid #ddd; padding: 20px; border-radius: 8px; margin-bottom: 15px; background: #fff; box-shadow: 0 2px 4px rgba(0,0,0,0.05);"
        >
          <h3 style="margin: 0 0 10px 0; color: #2c3e50;">{{ tarea.title }}</h3>
          
          <p style="color: #666; margin: 0 0 15px 0; white-space: pre-wrap;">
            {{ tarea.description || 'Sin descripción detallada.' }}
          </p>
          
          <div v-if="tarea.due_at" style="display: inline-block; padding: 5px 10px; background-color: #ffebee; color: #d32f2f; border-radius: 4px; font-size: 0.85rem; font-weight: bold;">
            📅 Fecha de entrega: {{ new Date(tarea.due_at).toLocaleDateString() }}
          </div>
        </li>
      </ul>

      <div v-else style="color: #777; padding: 30px; background: #f1f8e9; border: 1px dashed #8bc34a; border-radius: 8px; text-align: center; margin-top: 20px;">
        <span style="font-size: 2rem; display: block; margin-bottom: 10px;">🎉</span>
        <strong>¡Genial!</strong> No hay ninguna tarea pendiente para esta asignatura ahora mismo.
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()

// Sacamos el ID del grupo de la URL
const grupoId = route.params.id

const tareas = ref([])
const loading = ref(true)

const cargarTareas = async () => {
  try {
    const res = await fetch(`/api/groups/${grupoId}/tasks`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token}`
      }
    })

    if (res.ok) {
      tareas.value = await res.json()
    } else {
      console.error("Error al cargar las tareas del grupo")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  cargarTareas()
})
</script>