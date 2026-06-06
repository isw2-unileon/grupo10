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
// 1. Importamos el nuevo plugin para poder hacer clic
import interactionPlugin from '@fullcalendar/interaction' 

// 2. Creamos la función que se dispara al hacer clic en un día
const alHacerClicEnUnDia = async (info: any) => {
  // Sacamos un pop-up nativo del navegador pidiendo el título
  const titulo = prompt(`¿Qué tutoría quieres crear para el día ${info.dateStr}?`)
  
  // Si el usuario le da a cancelar o no escribe nada, salimos
  if (!titulo) return

  // Si escribió algo, preparamos el paquete para Go (igual que tu curl)
  const nuevoEvento = {
    owner_id: "2f51085c-16f7-4d74-b97c-4fb23e2d13c1", // Tu UUID real robado
    title: titulo,
    starts_at: info.dateStr + "T10:00:00Z", // Le ponemos las 10:00 por defecto
    ends_at: info.dateStr + "T11:00:00Z"    // Y que acabe a las 11:00
  }

  try {
    // 3. Hacemos el POST desde Vue (lo que hacía tu curl)
    const response = await fetch('/api/tutorings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(nuevoEvento)
    })

    if (response.ok) {
      // 4. Si Go dice que OK, volvemos a pedir los eventos para que se refresque la pantalla
      cargarEventosDeBD()
    } else {
      alert("Error al crear la tutoría en el servidor")
    }
  } catch (error) {
    console.error("Error de conexión:", error)
  }
}

// 5. Añadimos el plugin y el evento click a la configuración del calendario
const calendarOptions = ref({
  plugins: [dayGridPlugin, interactionPlugin], // <-- Añadimos el plugin aquí
  initialView: 'dayGridMonth',
  events: [],
  locale: 'es',
  firstDay: 1,
  dateClick: alHacerClicEnUnDia,               // <-- Le enchufamos nuestra función
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