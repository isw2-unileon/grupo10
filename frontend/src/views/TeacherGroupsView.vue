<template>
  <div class="groups-container" style="max-width: 800px; margin: 0 auto; padding: 20px;">
    <h2>👨‍🏫 Panel de Control de Asignaturas</h2>
    <p>Crea y gestiona tus grupos de alumnos.</p>

    <div style="background: #f3f4f6; padding: 20px; border-radius: 8px; margin-bottom: 30px;">
      <h3>Crear Nuevo Grupo</h3>
      <div style="display: flex; gap: 10px;">
        <input 
          v-model="nuevoGrupo" 
          type="text" 
          placeholder="Ej: Ingeniería del Software II"
          style="flex: 1; padding: 8px; border: 1px solid #ccc; border-radius: 4px;"
        />
        <button 
          @click="crearGrupo"
          style="padding: 8px 16px; background-color: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer;"
        >
          Crear
        </button>
      </div>
    </div>

    <div>
      <h3>Mis Grupos Actuales</h3>
      
      <ul v-if="grupos.length > 0" style="list-style: none; padding: 0;">
        <li 
          v-for="grupo in grupos" 
          :key="grupo.id"
          style="border: 1px solid #ddd; padding: 15px; border-radius: 8px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center;"
        >
          <strong>{{ grupo.name }}</strong>
          <button 
            @click="verDetalles(grupo.id)"
            style="padding: 6px 12px; background-color: #008CBA; color: white; border: none; border-radius: 4px; cursor: pointer;"
          >
            Gestionar Alumnos 👥
          </button>
        </li>
      </ul>
      
      <div v-else style="color: #666; font-style: italic;">
        No tienes ningún grupo creado todavía. ¡Anímate a crear el primero!
      </div>
    </div>
  </div>
</template>

<script setup>
import { API_BASE } from '@/services/apiBase'
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth' 
import { useRouter } from 'vue-router'
const router = useRouter()
// Inicializamos la tienda de autenticación
const auth = useAuthStore()

// Variables reactivas
const grupos = ref([])
const nuevoGrupo = ref('')

// Función para pedirle a Go los grupos de este profesor
const cargarGrupos = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/groups`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // 🔑 AQUÍ LE PASAMOS EL TOKEN A GO
        'Authorization': `Bearer ${auth.token}` 
      }
    })
    
    if (res.ok) {
      grupos.value = await res.json()
    } else if (res.status === 401) {
      alert("Sesión expirada o no autorizado. ¡Prueba a loguearte otra vez como profe! 👨‍🏫")
    } else {
      console.error("No se pudieron cargar los grupos")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  }
}

// Función para mandar el nuevo grupo a Go
const crearGrupo = async () => {
  if (!nuevoGrupo.value.trim()) {
    alert("¡Escribe un nombre para el grupo primero! 🛑")
    return
  }

  try {
    const res = await fetch(`${API_BASE}/api/groups`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        // 🔑 AQUÍ TAMBIÉN LE PASAMOS EL TOKEN A GO
        'Authorization': `Bearer ${auth.token}`
      },
      body: JSON.stringify({ name: nuevoGrupo.value })
    })

    if (res.ok) {
      alert("¡Grupo creado con éxito! 🎉")
      nuevoGrupo.value = '' // Vaciamos el input
      cargarGrupos()        // Recargamos la lista para que aparezca el nuevo
    } else {
      const errorData = await res.json().catch(() => ({}))
      alert("Error al crear: " + (errorData.error || "Fallo en el servidor"))
    }
  } catch (error) {
    console.error("Error al conectar:", error)
  }
}

// Función preparatoria para la siguiente fase
const verDetalles = (id) => {
    router.push(`/teacher/groups/${id}`)
  
}

// Cuando la pantalla nace, cargamos los grupos automáticamente
onMounted(() => {
  cargarGrupos()
})
</script>