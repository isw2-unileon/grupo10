<template>
  <div class="student-groups-container" style="max-width: 800px; margin: 0 auto; padding: 20px;">
    <h2>🎓 Mis Asignaturas y Grupos</h2>
    <p>Aquí puedes ver los grupos de clase a los que perteneces.</p>

    <div v-if="grupos.length > 0" style="margin-top: 20px;">
      <ul style="list-style: none; padding: 0;">
        <li 
          v-for="grupo in grupos" 
          :key="grupo.id"
          style="border: 1px solid #ddd; padding: 20px; border-radius: 8px; margin-bottom: 15px; background-color: #fafafa; display: flex; justify-content: space-between; align-items: center;"
        >
          <div>
            <h3 style="margin: 0 0 5px 0; color: #333;">{{ grupo.name }}</h3>
            <span style="font-size: 0.85rem; color: #666;">ID del Grupo: {{ grupo.id }}</span>
          </div>
          <button 
            @click="verTareasGrupo(grupo.id)"
            style="padding: 8px 16px; background-color: #ff9800; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: bold;"
          >
            Ver Tareas 📝
          </button>
        </li>
      </ul>
    </div>

    <div v-else style="margin-top: 40px; text-align: center; padding: 40px; border: 2px dashed #ccc; border-radius: 12px; background: #fff;">
      <div style="font-size: 3rem; margin-bottom: 15px;">⏳</div>
      <h3 style="color: #555;">Esperando a que te añadan a un grupo</h3>
      <p style="color: #777; max-width: 500px; margin: 0 auto;">
        Tu cuenta está activa, pero aún no apareces en ninguna asignatura. Dile a tu profesor que te añada utilizando el correo electrónico de tu cuenta.
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'

import { useRouter } from 'vue-router'
const router = useRouter()

const auth = useAuthStore()
const grupos = ref([])

// Función para pedirle a Go los grupos del alumno logueado
const cargarMisGrupos = async () => {
  try {
    const res = await fetch('/api/me/groups', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token}` // Pasamos el token del alumno
      }
    })

    if (res.ok) {
      grupos.value = await res.json()
    } else {
      console.error("Error al obtener los grupos del alumno")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  }
}

const verTareasGrupo = (id) => {
  router.push(`/student/groups/${id}/tasks`)
}

onMounted(() => {
  cargarMisGrupos()
})
</script>