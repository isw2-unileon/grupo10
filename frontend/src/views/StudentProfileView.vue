<template>
  <div style="max-width: 800px; margin: 40px auto; padding: 20px; font-family: sans-serif;">
    <router-link to="/student/groups" style="text-decoration: none; color: #4f46e5; font-weight: bold;">
      ⬅️ Volver a mis asignaturas
    </router-link>

    <div v-if="loading" style="text-align: center; margin-top: 40px;"><h3>Cargando tu expediente... ⏳</h3></div>
    
    <div v-else-if="profile">
      <!-- BASIC IDENTIFICATION CARD -->
      <div style="background: white; padding: 30px; border-radius: 12px; border: 1px solid #cbd5e1; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); margin-top: 20px; text-align: center;">
        <div style="font-size: 4rem; margin-bottom: 10px;">🎓</div>
        <h2 style="margin: 0; color: #1e293b; font-size: 2rem; text-transform: capitalize;">{{ getName(profile.email) }}</h2>
        <p style="color: #475569; font-size: 1.1rem; margin-top: 5px;">{{ profile.email }}</p>
        <span style="background: #e2e8f0; color: #334155; padding: 4px 12px; border-radius: 20px; font-size: 0.85rem; font-weight: bold; text-transform: uppercase;">
          Rol: Estudiante Registrado
        </span>
      </div>

      <h3 style="margin-top: 40px; color: #0f172a; border-bottom: 2px solid #e2e8f0; padding-bottom: 10px;">📊 Mi Expediente Académico</h3>

      <div v-if="!profile.analytics || profile.analytics.length === 0" style="text-align: center; padding: 40px; color: #64748b; background: #f8fafc; border-radius: 8px; border: 2px dashed #cbd5e1;">
        <span style="font-size: 2rem;">📚</span>
        <p style="margin-top: 10px;">No estás matriculado en ninguna asignatura todavía.</p>
      </div>

      <!-- COURSES LOOP -->
      <div v-for="course in (profile.analytics || [])" :key="course.group_id" style="background: white; border: 1px solid #cbd5e1; border-radius: 8px; margin-bottom: 20px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.05);">
        <div style="background: #f8fafc; padding: 15px 20px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #cbd5e1;">
          <h4 style="margin: 0; color: #0f172a; font-size: 1.2rem;">📘 {{ course.group_name }}</h4>
          <span :style="{ background: Number(course.total_average||0) >= 5 ? '#166534' : '#991b1b', color: 'white', padding: '6px 12px', borderRadius: '6px', fontWeight: 'bold' }">
            Media Asignatura: {{ Number(course.total_average || 0).toFixed(2) }}
          </span>
        </div>
        
        <div style="padding: 20px;">
          <h5 style="margin: 0 0 15px 0; color: #475569;">Progreso por Temas:</h5>
          
          <div v-if="!course.sections || course.sections.length === 0" style="color: #94a3b8; font-style: italic; font-size: 0.9rem;">
            El profesor no ha creado el temario todavía.
          </div>

          <div v-for="section in (course.sections || [])" :key="section.section_id" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; padding: 12px; background: #f1f5f9; border-radius: 6px; border: 1px solid #e2e8f0;">
            <div>
              <strong style="color: #1e293b; display: block; font-size: 1rem;">📁 {{ section.section_title }}</strong>
              <span style="color: #64748b; font-size: 0.85rem;">
                {{ section.graded_count > 0 ? `${section.graded_count} evaluaciones corregidas` : 'Ninguna tarea evaluada aún' }}
              </span>
            </div>
            <div style="font-size: 1.4rem; font-weight: bold;" :style="{ color: Number(section.average || 0) >= 5 ? '#166534' : '#991b1b' }">
              {{ Number(section.average || 0).toFixed(2) }}
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { API_BASE } from '@/services/apiBase'
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const profile = ref(null)
const loading = ref(true)

// Helper to extract name from email (e.g. carlos@unileon.es -> Carlos)
const getName = (email) => {
  if (!email) return "Estudiante"
  return email.split('@')[0]
}

const loadProfile = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/me/profile`, {
      headers: { 'Authorization': `Bearer ${auth.token}` }
    })
    if (res.ok) profile.value = await res.json()
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

onMounted(() => { loadProfile() })
</script>