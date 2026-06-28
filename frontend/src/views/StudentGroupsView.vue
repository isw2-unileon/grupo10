<template>
  <div class="student-groups-container" style="max-width: 800px; margin: 0 auto; padding: 20px; font-family: sans-serif;">
    
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; border-bottom: 2px solid #e2e8f0; padding-bottom: 15px;">
      <div>
        <h2 style="margin: 0; color: #1e293b; font-size: 2rem;">🎓 Mis Asignaturas y Grupos</h2>
        <p style="margin: 5px 0 0 0; color: #64748b; font-size: 0.95rem;">Aquí puedes ver los grupos de clase a los que perteneces.</p>
      </div>
      <div style="display: flex; gap: 10px;">
        <button @click="$router.push('/student/ai-tutor')" style="background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); color: white; padding: 10px 20px; border: none; border-radius: 8px; font-weight: bold; cursor: pointer; font-size: 1rem; box-shadow: 0 4px 6px -1px rgba(99,102,241,0.2); transition: transform 0.2s;">
          🤖 Tutor IA de Refuerzo
        </button>
        <button @click="$router.push('/student/profile')" style="background: #4f46e5; color: white; padding: 10px 20px; border: none; border-radius: 8px; font-weight: bold; cursor: pointer; font-size: 1rem; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); transition: transform 0.2s;">
          👤 Ver Mi Perfil y Notas
        </button>
      </div>
    </div>

    <div v-if="grupos.length > 0">
      <ul style="list-style: none; padding: 0;">
        <li 
          v-for="grupo in grupos" 
          :key="grupo.id"
          style="border: 1px solid #cbd5e1; padding: 20px; border-radius: 8px; margin-bottom: 15px; background-color: #ffffff; box-shadow: 0 2px 4px rgba(0,0,0,0.05); display: flex; justify-content: space-between; align-items: center;"
        >
          <div>
            <h3 style="margin: 0 0 5px 0; color: #0f172a;">{{ grupo.name }}</h3>
            <span style="font-size: 0.85rem; color: #64748b;">ID del Grupo: {{ grupo.id }}</span>
          </div>
          <button 
            @click="verTareasGrupo(grupo.id)"
            style="padding: 10px 18px; background-color: #d97706; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: bold; font-size: 0.95rem;"
          >
            Entrar al Campus 📝
          </button>
        </li>
      </ul>
    </div>

    <div v-else style="margin-top: 40px; text-align: center; padding: 40px; border: 2px dashed #cbd5e1; border-radius: 12px; background: #f8fafc;">
      <div style="font-size: 3rem; margin-bottom: 15px;">⏳</div>
      <h3 style="color: #334155;">Esperando a que te añadan a un group</h3>
      <p style="color: #64748b; max-width: 500px; margin: 0 auto; line-height: 1.5;">
        Tu cuenta está activa, pero aún no apareces matriculado en ninguna asignatura. Dile a tu profesor que te añada utilizando tu correo electrónico.
      </p>
    </div>
  </div>
</template>

<script setup>
import { API_BASE } from '@/services/apiBase'
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const router = useRouter()
const auth = useAuthStore()
const grupos = ref([])

// Función para pedirle a Go los grupos del alumno logueado
const cargarMisGrupos = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/me/groups`, {
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