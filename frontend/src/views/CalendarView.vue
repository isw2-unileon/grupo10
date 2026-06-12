<template>
  <div class="calendar-container">
    <h1>Mi Calendario 📅</h1>
    
    <div class="calendar-wrapper">
      <FullCalendar :options="calendarOptions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import FullCalendar from '@fullcalendar/vue3'
import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin from '@fullcalendar/interaction'
// 1. Importamos la tienda del usuario de tu compi
import { useAuthStore } from '@/stores/auth' 

// 2. Inicializamos la tienda
const auth = useAuthStore()

const alHacerClicEnUnDia = async (info: any) => {
  // 3. ¡EL GUARDIA DE SEGURIDAD! Si no es profe, lo echamos.
  if (auth.user?.role !== 'teacher') {
    alert("¡Quieto ahí! Solo los profesores pueden crear tutorías. 🛑")
    return
  }

  const titulo = prompt(`¿Qué tutoría quieres crear para el día ${info.dateStr}?`)
  
  if (!titulo) return

  // 4. Usamos el ID REAL del profesor que ha iniciado sesión
  const nuevoEvento = {
    owner_id: auth.user.id, // <-- ¡Adiós al UUID robado!
    title: titulo,
    starts_at: info.dateStr + "T10:00:00Z",
    ends_at: info.dateStr + "T11:00:00Z"
  }

  try {
    const response = await fetch('/api/tutorings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(nuevoEvento)
    })

    if (response.ok) {
      cargarEventosDeBD()
    } else {
      alert("Error al crear la tutoría en el servidor")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  }
}
const alHacerClicEnUnEvento = async (info: any) => {
  // 1. ¡Control de seguridad! Solo los alumnos reservan
  if (auth.user?.role !== 'student') {
    alert("Los profesores no pueden reservar tutorías, ¡que son los que las dan! 👨‍🏫")
    return
  }

  

  // 2. Preguntamos confirmación al alumno
  const confirmar = confirm(`¿Quieres reservar la tutoría "${info.event.title}"?`)
  if (!confirmar) return

  // 3. Preparamos los datos para el endpoint de reservar que hizo tu compi
  const datosReserva = {
    event_id: info.event.id,  // El ID de la tutoría que viene de la BD
    student_id: auth.user.id     // El ID del alumno logueado
  }

  try {
    // 4. Lanzamos la petición POST a /book
    const response = await fetch('/api/tutorings/book', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(datosReserva)
    })

    if (response.ok) {
      alert("¡Tutoría reservada con éxito! 📅🎉")
      // Refrescamos el calendario para que se vean los cambios
      cargarEventosDeBD()
    } else {
      const errorData = await response.json().catch(() => ({}))
      alert(errorData.message || "Error al procesar la reserva en el servidor")
    }
  } catch (error) {
    console.error("Error al conectar con el servidor:", error)
  }
}

// 5. Añadimos el plugin y el evento click a la configuración del calendario
const calendarOptions = ref({
  plugins: [dayGridPlugin, interactionPlugin],
  initialView: 'dayGridMonth',
  events: [],
  locale: 'es',
  firstDay: 1,
  dateClick: alHacerClicEnUnDia,       
  eventClick: alHacerClicEnUnEvento,   
  headerToolbar: {
    left: 'prev,next today',
    center: 'title',
    right: 'dayGridMonth'
  }
})

// 3. Función asíncrona para pedir datos a Go
const cargarEventosDeBD = async () => {
  try {
    // 1. Corregimos la ruta para que coincida con Go
    const response = await fetch('/api/tutorings') 
    
    if (!response.ok) {
      throw new Error('Fallo al conectar con Go')
    }

    const datosDeGo = await response.json()
    
    // 2. TRADUCCIÓN: Adaptamos los nombres de Go a los de FullCalendar
    const eventosParaCalendario = datosDeGo.map((evento: any) => ({
        id: evento.id,
        title: evento.title,
        start: evento.starts_at, // FullCalendar necesita "start"
        end: evento.ends_at,     // FullCalendar necesita "end"
        color: '#42b883'         // Les ponemos color verde Vue
    }))
    
    // 3. Metemos los datos traducidos en el calendario
    calendarOptions.value.events = eventosParaCalendario

  } catch (error) {
    console.error("🚨 Error pidiendo eventos al servidor:", error)
  }
}

// 5. Le decimos a Vue: "En cuanto la pantalla cargue, ejecuta esta función"
onMounted(() => {
  console.log("El componente ha nacido. Voy a pedir los eventos...")
  cargarEventosDeBD()
})
</script>

<style scoped>
/* Le damos un poco de estilo al fondo para que resalte */
.calendar-container {
  padding: 20px;
  max-width: 900px; /* Lo hacemos más ancho para que quepa bien el mes */
  margin: 0 auto;
}

.calendar-wrapper {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.05);
}

/* Hacemos que el título quede centrado y bonito */
h1 {
  text-align: center;
  color: #2c3e50;
  margin-bottom: 20px;
}
</style>