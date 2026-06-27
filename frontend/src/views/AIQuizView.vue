<template>
  <div style="max-width: 800px; margin: 40px auto; padding: 20px; font-family: sans-serif;">
    <router-link to="/student/groups" style="text-decoration: none; color: #4f46e5; font-weight: bold; display: inline-block; margin-bottom: 20px;">
      ⬅️ Volver a mis asignaturas
    </router-link>

    <div style="background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%); color: white; padding: 35px; border-radius: 16px; margin-bottom: 30px; box-shadow: 0 10px 25px -5px rgba(79, 70, 229, 0.3);">
      <h2 style="margin: 0; font-size: 2.2rem;">🧠 Generador de Cuestionarios Cruzados con IA</h2>
      <p style="margin: 8px 0 0 0; opacity: 0.9; font-size: 1.05rem;">Diseña tu propio examen de entrenamiento seleccionando múltiples temas y asignaturas de forma simultánea.</p>
    </div>

    <div v-if="loadingInit" style="text-align: center; padding: 40px;">
      <h3>Estructurando tu mapa de conocimiento... 🔄</h3>
    </div>

    <div v-else-if="!cuestionario">
      <h3 style="color: #1e293b; margin-bottom: 15px;">1. Selecciona los bloques temáticos de refuerzo:</h3>
      
      <div v-for="asignatura in asignaturas" :key="asignatura.id" style="background: white; border: 1px solid #cbd5e1; border-radius: 10px; margin-bottom: 20px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
        <div style="background: #f8fafc; padding: 15px 20px; border-bottom: 1px solid #cbd5e1; display: flex; justify-content: space-between; align-items: center;">
          <h4 style="margin: 0; color: #0f172a; font-size: 1.15rem;">📘 Asignatura: {{ asignatura.name }}</h4>
          <small style="color: #64748b; font-weight: bold;">ID: {{ asignatura.id }}</small>
        </div>
        
        <div style="padding: 15px 20px;">
          <div v-if="!temasPorAsignatura[asignatura.id] || temasPorAsignatura[asignatura.id].length === 0" style="color: #94a3b8; font-style: italic; font-size: 0.9rem;">
            Esta materia no tiene bloques temáticos configurados por el profesor.
          </div>
          <div v-else style="display: grid; grid-template-columns: 1fr; gap: 10px;">
            <label v-for="tema in temasPorAsignatura[asignatura.id]" :key="tema.id" style="display: flex; align-items: center; gap: 12px; padding: 12px; background: #f1f5f9; border-radius: 6px; cursor: pointer; transition: background 0.2s;">
              <input type="checkbox" :value="tema.id" v-model="selectedThemes" style="width: 18px; height: 18px; accent-color: #4f46e5;" />
              <span style="color: #334155; font-weight: 500;">📁 {{ tema.title }}</span>
            </label>
          </div>
        </div>
      </div>

      <div style="margin-top: 30px; background: #f8fafc; border: 1px solid #e2e8f0; padding: 20px; border-radius: 12px; display: flex; justify-content: space-between; align-items: center;">
        <div>
          <span style="font-weight: bold; color: #1e293b; display: block; font-size: 1.1rem;">Temas seleccionados: {{ selectedThemes.length }}</span>
          <small style="color: #64748b;">La IA balanceará las preguntas de forma equitativa.</small>
        </div>
        <button @click="solicitarCuestionarioIA" :disabled="selectedThemes.length === 0 || generando" style="background: #10b981; color: white; border: none; padding: 14px 28px; border-radius: 8px; font-weight: bold; cursor: pointer; font-size: 1rem; box-shadow: 0 4px 12px rgba(16,185,129,0.2);">
          {{ generando ? 'Procesando Inteligencia... ⏳' : 'Generar Macro-Test de IA 🚀' }}
        </button>
      </div>
    </div>

    <div v-if="cuestionario && !generando" style="margin-top: 10px;">
      <div style="background: #f5f3ff; border: 1px solid #ddd; padding: 15px; border-radius: 8px; margin-bottom: 25px;">
        <h3 style="margin: 0; color: #4f46e5;">📋 {{ cuestionario.section_title }}</h3>
      </div>
      
      <div v-for="(q, qIdx) in cuestionario.questions" :key="qIdx" style="background: white; border: 1px solid #cbd5e1; padding: 20px; border-radius: 10px; margin-bottom: 25px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
        <p style="font-weight: bold; color: #0f172a; margin: 0 0 15px 0; font-size: 1.1rem;">{{ qIdx + 1 }}. {{ q.question_text }}</p>
        
        <div style="display: flex; flex-direction: column; gap: 10px;">
          <label v-for="(opt, oIdx) in q.options" :key="oIdx" style="padding: 12px; border: 1px solid #e2e8f0; border-radius: 6px; display: flex; align-items: center; gap: 12px; cursor: pointer;" :style="obtenerEstiloOpcion(qIdx, oIdx, opt)">
            <input type="radio" :name="`ai-q-${qIdx}`" :disabled="testEnviado" @change="respuestasSeleccionadas[qIdx] = oIdx" />
            <span>{{ opt.text }}</span>
          </label>
        </div>

        <div v-if="testEnviado" style="margin-top: 15px; padding: 12px; background: #f8fafc; border-left: 4px solid #7c3aed; font-size: 0.9rem; color: #475569; line-height: 1.5;">
          💡 <strong>Retroalimentación Pedagógica IA:</strong> {{ q.explanation }}
        </div>
      </div>

      <button v-if="!testEnviado" @click="corregirTestLocalmente" :disabled="Object.keys(respuestasSeleccionadas).length < cuestionario.questions.length" style="background: #4f46e5; color: white; border: none; padding: 16px; width: 100%; border-radius: 8px; font-weight: bold; font-size: 1.1rem; cursor: pointer; box-shadow: 0 4px 12px rgba(79,70,229,0.2);">
        Finalizar Examen y Calificar
      </button>

      <div v-else style="background: #f0fdf4; border: 1px solid #bbf7d0; padding: 25px; border-radius: 12px; text-align: center; margin-top: 30px; box-shadow: 0 4px 10px rgba(0,0,0,0.02);">
        <h4 style="margin: 0; color: #166534; font-size: 1.4rem;">¡Entrenamiento de Cobertura Finalizado!</h4>
        <p style="font-size: 2.2rem; font-weight: bold; margin: 12px 0; color: #14532d;">Calificación Obtenida: {{ notaFinal }}/10</p>
        <button @click="reiniciarModulo" style="background: #4f46e5; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-weight: bold;">Diseñar otra simulación</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const asignaturas = ref([])
const temasPorAsignatura = ref({})
const selectedThemes = ref([])

const loadingInit = ref(true)
const generando = ref(false)
const cuestionario = ref(null)

const respuestasSeleccionadas = ref({})
const testEnviado = ref(false)
const notaFinal = ref(0)

// Carga masiva paralela en el onMounted para optimizar el rendimiento de la UI
const cargarEcosistemaEstudiante = async () => {
  try {
    loadingInit.value = true
    const res = await fetch('/api/me/groups', { headers: { 'Authorization': `Bearer ${auth.token}` } })
    if (res.ok) {
      asignaturas.value = await res.json()
      // Ejecutamos en paralelo la descarga de todos los temas de todas tus asignaturas juntas
      await Promise.all(asignaturas.value.map(async (g) => {
        const resT = await fetch(`/api/groups/${g.id}/content`, { headers: { 'Authorization': `Bearer ${auth.token}` } })
        if (resT.ok) {
          temasPorAsignatura.value[g.id] = await resT.json() || []
        }
      }))
    }
  } catch (e) {
    console.error(e)
  } finally {
    loadingInit.value = false
  }
}

const solicitarCuestionarioIA = async () => {
  generando.value = true
  cuestionario.value = null
  testEnviado.value = false
  respuestasSeleccionadas.value = {}
  try {
    // Inyectamos la lista de identificadores separados por comas
    const queryIds = selectedThemes.value.join(',')
    const res = await fetch(`/api/ai-quiz?sections=${queryIds}`, { headers: { 'Authorization': `Bearer ${auth.token}` } })
    if (res.ok) cuestionario.value = await res.json()
  } catch (e) { console.error(e) }
  finally { generando.value = false }
}

const corregirTestLocalmente = () => {
  let aciertos = 0
  cuestionario.value.questions.forEach((q, idx) => {
    const seleccion = respuestasSeleccionadas.value[idx]
    if (seleccion !== undefined && q.options[seleccion].is_correct) aciertos++
  })
  notaFinal.value = (aciertos / cuestionario.value.questions.length) * 10
  testEnviado.value = true
}

const obtenerEstiloOpcion = (qIdx, oIdx, opt) => {
  if (!testEnviado.value) {
    return respuestasSeleccionadas.value[qIdx] === oIdx ? { background: '#f5f3ff', borderColor: '#6366f1' } : {}
  }
  if (opt.is_correct) return { background: '#dcfce3', borderColor: '#22c55e', color: '#15803d', fontWeight: 'bold' }
  if (respuestasSeleccionadas.value[qIdx] === oIdx && !opt.is_correct) return { background: '#fee2e2', borderColor: '#ef4444', color: '#b91c1c' }
  return { opacity: 0.5 }
}

const reiniciarModulo = () => {
  cuestionario.value = null
  testEnviado.value = false
  respuestasSeleccionadas.value = {}
  selectedThemes.value = []
}

onMounted(() => { cargarEcosistemaEstudiante() })
</script>