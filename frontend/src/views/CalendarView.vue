<script setup lang="ts">
import { API_BASE } from '@/services/apiBase'
import { ref, onMounted } from 'vue'
import FullCalendar from '@fullcalendar/vue3'
import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin from '@fullcalendar/interaction'
import timeGridPlugin from '@fullcalendar/timegrid'

import { useAuthStore } from '@/stores/auth' 

const auth = useAuthStore()

// NEW FUNCTION: Handles drag and drop selection for tutoring slots
const handleRangeSelect = async (info: any) => {
  if (auth.user?.role !== 'teacher') {
    alert("¡Quieto ahí! Solo los profesores pueden crear tutorías. 🛑")
    return
  }

  // 1. Request Title and Description
  const title = prompt("¿Qué título le ponemos a la tutoría?")
  if (!title) return // Exit if cancelled

  const description = prompt("Añade los detalles o descripción (opcional):")

  // 2. Map FullCalendar timestamps to backend payload
  const newEvent = {
    owner_id: auth.user.id,
    title: title,
    description: description || "", 
    starts_at: info.startStr,       
    ends_at: info.endStr            
  }

  try {
    const response = await fetch(`${API_BASE}/api/tutorings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newEvent)
    })

    // 3. Error handling from backend (e.g. time travel protection)
    if (!response.ok) {
      const errorText = await response.text()
      alert(`⚠️ No se pudo crear la tutoría:\n${errorText}`)
      return
    }

    alert("¡Tutoría creada con éxito! 📅🎉")
    fetchEventsFromDB()
  } catch (error) {
    console.error("Connection error:", error)
    alert("🚨 Error de red al conectar con el servidor.")
  }
}

const handleEventClick = async (info: any) => {
  if (auth.user?.role !== 'student') {
    alert("Los profesores no pueden reservar tutorías, ¡que son los que las dan! 👨‍🏫")
    return
  }

  const shouldConfirm = confirm(`¿Quieres reservar la tutoría "${info.event.title}"?`)
  if (!shouldConfirm) return

  const bookingData = {
    event_id: info.event.id,
    student_id: auth.user.id
  }

  try {
    const response = await fetch(`${API_BASE}/api/tutorings/book`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(bookingData)
    })

    if (response.ok) {
      alert("¡Tutoría reservada con éxito! 📅🎉")
      fetchEventsFromDB()
    } else {
      const errorData = await response.json().catch(() => ({}))
      alert(`⚠️ ${errorData.message || "Error al procesar la reserva en el servidor"}`)
    }
  } catch (error) {
    console.error("Server connection error:", error)
  }
}

const fetchEventsFromDB = async () => {
  try {
    const response = await fetch(`${API_BASE}/api/tutorings`) 
    if (!response.ok) throw new Error('Fallo al conectar con Go')

    const eventsData = await response.json()
    
    calendarOptions.value.events = eventsData.map((event: any) => ({
        id: event.id,
        title: event.title,
        start: event.starts_at, 
        end: event.ends_at,     
        color: '#42b883',
        extendedProps: {
          description: event.description
        }
    }))
  } catch (error) {
    console.error("🚨 Error pidiendo eventos al servidor:", error)
  }
}

const calendarOptions = ref({
  plugins: [dayGridPlugin, interactionPlugin, timeGridPlugin],
  initialView: 'timeGridWeek',         
  selectable: true,                    
  select: handleRangeSelect,           
  eventClick: handleEventClick,   
  events: [],
  locale: 'es',
  firstDay: 1,
  slotMinTime: '08:00:00',             
  slotMaxTime: '21:00:00',             
  headerToolbar: {
    left: 'prev,next today',
    center: 'title',
    right: 'dayGridMonth,timeGridWeek,timeGridDay' 
  }
})

onMounted(() => {
  fetchEventsFromDB()
})
</script>

<template>
  <div class="calendar-container">
    <h1 class="calendar-title">📅 Calendario de Tutorías</h1>
    <FullCalendar :options="calendarOptions" />
  </div>
</template>

<style scoped>
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

:deep(.fc-event) {
  cursor: pointer; 
  transition: transform 0.2s;
}

:deep(.fc-event:hover) {
  transform: scale(1.02); 
}

:deep(.fc-toolbar-title) {
  text-transform: capitalize; 
}
</style>