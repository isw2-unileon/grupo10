<template>
  <div style="max-width: 800px; margin: 40px auto; padding: 20px; font-family: sans-serif;">
    <button @click="$router.back()" style="background: none; border: none; color: #4f46e5; cursor: pointer; font-weight: bold; margin-bottom: 20px; font-size: 1rem;">
      ⬅️ Volver al Campus Virtual
    </button>

    <div v-if="loading" style="text-align: center;"><h3>Cargando cuestionario... ⏳</h3></div>
    
    <div v-else-if="quiz" style="background: white; padding: 30px; border-radius: 12px; border: 1px solid #cbd5e1; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);">
      
      <div v-if="modoRevision" style="background: #eff6ff; border: 2px solid #3b82f6; padding: 20px; border-radius: 8px; margin-bottom: 25px; text-align: center;">
        <h3 style="margin: 0; color: #1d4ed8; font-size: 1.4rem;">Cuestionario Completado</h3>
        <p style="font-size: 1.8rem; font-weight: bold; margin: 10px 0; color: #1e3a8a;">
          🎯 Nota Obtenida: {{ quiz.current_grade !== undefined ? quiz.current_grade : notaInmediata }}/10
        </p>
        <span style="color: #475569; font-size: 0.9rem;">Las respuestas correctas e incorrectas se detallan abajo.</span>
      </div>

      <h2 style="margin-top: 0; color: #1e293b;">📝 Cuestionario: {{ quiz.title }}</h2>
      <p style="color: #475569; margin-bottom: 30px; font-style: italic;">Evaluación de contenidos obligatoria.</p>

      <div v-for="(pregunta, index) in quiz.questions" :key="pregunta.id" style="margin-bottom: 25px; padding: 18px; background: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0;">
        <h4 style="margin: 0 0 15px 0; color: #0f172a; font-size: 1.1rem;">{{ index + 1 }}. {{ pregunta.question_text }}</h4>
        
        <div 
          v-for="opcion in pregunta.options" 
          :key="opcion.id" 
          style="margin-bottom: 8px; display: flex; align-items: center; gap: 10px; padding: 8px; border-radius: 6px; transition: background 0.2s;"
          :style="obtenerEstiloFilaRevision(opcion)"
        >
          <input 
            type="radio" 
            :name="`pregunta-${pregunta.id}`" 
            :id="`opt-${opcion.id}`"
            :value="opcion.id"
            v-model="respuestas[pregunta.id]"
            :disabled="modoRevision"
            style="transform: scale(1.2);"
          />
          <label :for="`opt-${opcion.id}`" style="cursor: pointer; color: #334155; font-size: 0.95rem; flex-grow: 1; display: flex; justify-content: space-between;">
            <span>{{ opcion.option_text }}</span>
            
            <span v-if="modoRevision && opcion.selected && opcion.is_correct" style="color: #166534; font-weight: bold;">(Tu Respuesta - Correcta) ✨</span>
            <span v-if="modoRevision && opcion.selected && !opcion.is_correct" style="color: #991b1b; font-weight: bold;">(Tu Respuesta - Incorrecta) ❌</span>
            <span v-if="modoRevision && !opcion.selected && opcion.is_correct" style="color: #166534; font-weight: bold;">(Opción Correcta) ✔️</span>
          </label>
        </div>
      </div>

      <button v-if="!modoRevision" @click="enviarCuestionario" style="width: 100%; padding: 14px; background: #4f46e5; color: white; border: none; border-radius: 6px; font-weight: bold; font-size: 1.1rem; cursor: pointer; margin-top: 10px;">
        Finalizar y Calcular Nota en el Acto 🚀
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const resourceId = route.params.resourceId

const quiz = ref(null)
const loading = ref(true)
const modoRevision = ref(false)
const respuestas = ref({})
const notaInmediata = ref(0)

function getHeaders() {
  return { 'Authorization': `Bearer ${auth.token}` }
}

const evaluarEntornoQuiz = async () => {
  try {
    loading.value = true
    
    // 1. Averiguar primero en el árbol general si el alumno ya lo había enviado previamente
    const resContent = await fetch(`/api/groups/${route.params.groupId}/content`, { headers: getHeaders() })
    if (resContent.ok) {
      const secciones = await resContent.json() || []
      // Buscamos si este recurso ya figura como entregado (has_submitted)
      for (const sec of secciones) {
        const recursoEncontrado = sec.resources?.find(r => r.id === resourceId)
        if (recursoEncontrado && recursoEncontrado.has_submitted) {
          modoRevision.value = true
        }
      }
    }

    // 2. Cargar el cuestionario según el modo detectado
    if (modoRevision.value) {
      // Si está en revisión, llamamos al nuevo endpoint relacional pasándole la ID del alumno logueado
      const res = await fetch(`/api/resources/${resourceId}/review/${auth.user.id}`, { headers: getHeaders() })
      if (res.ok) quiz.value = await res.json()
    } else {
      // Si va a responder por primera vez, cargamos el esqueleto del quiz normal
      const res = await fetch(`/api/resources/${resourceId}/quiz`, { headers: getHeaders() })
      if (res.ok) quiz.value = await res.json()
    }
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

const enviarCuestionario = async () => {
  const totalPreguntas = quiz.value.questions?.length || 0
  if (Object.keys(respuestas.value).length < totalPreguntas) {
    if (!confirm("No has respondido a todas las preguntas. ¿Quieres entregar el test de todas formas?")) return
  }
  
  try {
    loading.value = true
    const res = await fetch(`/api/resources/${resourceId}/submit-quiz`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${auth.token}` },
      body: JSON.stringify({ answers: respuestas.value })
    })
    
    if (res.ok) {
      const data = await res.json()
      notaInmediata.value = data.grade
      alert(`¡Cuestionario guardado con éxito! Nota calculada instantáneamente: ${data.grade}/10`)
      
      // Forzamos el cambio automático a modo revisión para que vea la corrección al segundo
      modoRevision.value = true
      const resRev = await fetch(`/api/resources/${resourceId}/review/${auth.user.id}`, { headers: getHeaders() })
      if (resRev.ok) quiz.value = await resRev.json()
    }
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

// Pinta las filas de opciones según aciertos o fallos en la revisión
const obtenerEstiloFilaRevision = (opcion) => {
  if (!modoRevision.value) return {}
  
  if (opcion.selected && opcion.is_correct) {
    return { background: '#dcfce3', border: '1px solid #166534' } // Verde si acertó
  }
  if (opcion.selected && !opcion.is_correct) {
    return { background: '#fee2e2', border: '1px solid #991b1b' } // Rojo si falló
  }
  if (!opcion.selected && opcion.is_correct) {
    return { background: '#fef08a', border: '1px dashed #ca8a04' } // Amarillo para indicar cuál era la buena
  }
  return {}
}

onMounted(() => { evaluarEntornoQuiz() })
</script>