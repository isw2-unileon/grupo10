<script setup lang="ts">
import { ref, onMounted } from 'vue'
import FullCalendar from '@fullcalendar/vue3'
import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin from '@fullcalendar/interaction'
import timeGridPlugin from '@fullcalendar/timegrid' // ⏱️ NUEVO: Para ver las horas del día

import { useAuthStore } from '@/stores/auth' 

const auth = useAuthStore()

// 🚀 NUEVA FUNCIÓN: Ahora usamos "select" al arrastrar en lugar de "dateClick"
const alSeleccionarRango = async (info: any) => {
  if (auth.user?.role !== 'teacher') {
    alert("¡Quieto ahí! Solo los profesores pueden crear tutorías. 🛑")
    return
  }

  // 1. Pedimos Título y Descripción
  const titulo = prompt("¿Qué título le ponemos a la tutoría?")
  if (!titulo) return // Si cancela, salimos

  const descripcion = prompt("Añade los detalles o descripción (opcional):")

  // 2. FullCalendar nos da las horas exactas que hemos arrastrado
  const nuevoEvento = {
    owner_id: auth.user.id,
    title: titulo,
    description: descripcion || "", // Mandamos la descripción a Go
    starts_at: info.startStr,       // Hora de inicio exacta del arrastre
    ends_at: info.endStr            // Hora de fin exacta del arrastre
  }

  try {
    const response = await fetch('/api/tutorings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(nuevoEvento)
    })

    // 3. ¡Capturamos los errores del backend (como el de viajar al pasado)!
    if (!response.ok) {
      const errorTexto = await response.text()
      alert(`⚠️ No se pudo crear la tutoría:\n${errorTexto}`)
      return
    }

    alert("¡Tutoría creada con éxito! 📅🎉")
    cargarEventosDeBD()
  } catch (error) {
    console.error("Error de conexión:", error)
    alert("🚨 Error de red al conectar con el servidor.")
  }
}

const alHacerClicEnUnEvento = async (info: any) => {
  if (auth.user?.role !== 'student') {
    alert("Los profesores no pueden reservar tutorías, ¡que son los que las dan! 👨‍🏫")
    return
  }

  const confirmar = confirm(`¿Quieres reservar la tutoría "${info.event.title}"?`)
  if (!confirmar) return

  const datosReserva = {
    event_id: info.event.id,
    student_id: auth.user.id
  }

  try {
    const response = await fetch('/api/tutorings/book', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(datosReserva)
    })

    if (response.ok) {
      alert("¡Tutoría reservada con éxito! 📅🎉")
      cargarEventosDeBD()
    } else {
      const errorData = await response.json().catch(() => ({}))
      alert(`⚠️ ${errorData.message || "Error al procesar la reserva en el servidor"}`)
    }
  } catch (error) {
    console.error("Error al conectar con el servidor:", error)
  }
}

const cargarEventosDeBD = async () => {
  try {
    const response = await fetch('/api/tutorings') 
    if (!response.ok) throw new Error('Fallo al conectar con Go')

    const datosDeGo = await response.json()
    
    calendarOptions.value.events = datosDeGo.map((evento: any) => ({
        id: evento.id,
        title: evento.title,
        start: evento.starts_at, 
        end: evento.ends_at,     
        color: '#42b883',
        // Podemos guardar la descripción dentro del evento para usarla luego si queréis
        extendedProps: {
          description: evento.description
        }
    }))
  } catch (error) {
    console.error("🚨 Error pidiendo eventos al servidor:", error)
  }
}

const calendarOptions = ref({
  // Añadimos el plugin de las horas
  plugins: [dayGridPlugin, interactionPlugin, timeGridPlugin],
  initialView: 'timeGridWeek',         // ⏱️ Empezamos viendo la semana con horas
  selectable: true,                    // 🖱️ ¡Activamos el arrastrar y soltar!
  select: alSeleccionarRango,          // Llamamos a la nueva función al arrastrar
  eventClick: alHacerClicEnUnEvento,   
  events: [],
  locale: 'es',
  firstDay: 1,
  slotMinTime: '08:00:00',             // El calendario empieza a las 8am
  slotMaxTime: '21:00:00',             // Y acaba a las 9pm
  headerToolbar: {
    left: 'prev,next today',
    center: 'title',
    // Añadimos los botones para cambiar entre mes, semana y día
    right: 'dayGridMonth,timeGridWeek,timeGridDay' 
  }
})

onMounted(() => {
  console.log("El componente ha nacido. Voy a pedir los eventos...")
  cargarEventosDeBD()
})
</script>
<template>
  <div class="calendar-container">
    <h1 class="calendar-title">📅 Calendario de Tutorías</h1>
    
    <FullCalendar :options="calendarOptions" />
  </div>
</template>

<style scoped>
/* Un poco de chapa y pintura para que no se vea feo ni ocupe el 200% de la pantalla */
.calendar-container {
  max-width: 1100px;
  margin: 0 auto;
  padding: 20px;
  background-color: #ffffff;
  border-radius: 12px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.05);
}

.calendar-title {
  text-align: center;
  color: #2c3e50;
  margin-bottom: 24px;
  font-size: 2rem;
  font-weight: bold;
}

/* Usamos :deep() porque los estilos de FullCalendar se generan dinámicamente 
  y si no, Vue los ignoraría al tener "scoped" 
*/
:deep(.fc-event) {
  cursor: pointer; /* Hace que salga la manita al pasar por encima de las tutorías */
  transition: transform 0.2s;
}

:deep(.fc-event:hover) {
  transform: scale(1.02); /* Efecto chulo al hacer hover */
}

:deep(.fc-toolbar-title) {
  text-transform: capitalize; /* Para que los meses salgan con la primera en mayúscula */
}
</style>