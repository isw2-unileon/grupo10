<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="portal-container">
    <header class="welcome-header">
      <h1>Panel de Docente 👨‍🏫</h1>
      <p>Hola, {{ auth.user?.email }}. Este es tu centro de control de asignaturas.</p>
    </header>

    <div class="features-grid">
      <section class="feature-card">
        <div class="icon">✅</div>
        <h2>Aprobación de Apuntes</h2>
        <p>Revisa los apuntes que tus alumnos han procesado con la IA y apruébalos para validar su calidad.</p>
        <RouterLink to="/teacher/dashboard" class="btn-action teacher-btn">Ver Apuntes Pendientes</RouterLink>
      </section>

      <section class="feature-card ">
        <div class="icon">📅</div>
        <h2>Gestión de Tutorías</h2>
        <p>Publica franjas horarias libres en tu calendario para que los alumnos puedan reservar citas contigo.</p>
        <RouterLink to="/calendar" class="btn-action teacher-btn">Gestionar Calendario</RouterLink>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* Puedes copiar exactamente los mismos estilos del StudentView.vue aquí */
.portal-container { max-width: 1100px; margin: 0 auto; padding: 2rem 1rem; }
.welcome-header { margin-bottom: 3rem; text-align: center; }
.welcome-header h1 { color: #1e293b; font-size: 2.5rem; margin-bottom: 0.5rem; }
.welcome-header p { color: #64748b; font-size: 1.1rem; }
.features-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 2rem; }
.feature-card { background: white; border: 1px solid #e2e8f0; border-radius: 0.75rem; padding: 2rem; display: flex; flex-direction: column; align-items: flex-start; box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1); }
.icon { font-size: 3rem; margin-bottom: 1rem; }
h2 { margin-bottom: 1rem; color: #334155; }
p { flex-grow: 1; margin-bottom: 1.5rem; color: #475569; line-height: 1.5; }
.btn-action { color: white; padding: 0.75rem 1.5rem; border-radius: 0.5rem; text-decoration: none; font-weight: bold; text-align: center; width: 100%; box-sizing: border-box; }
.teacher-btn { background-color: #0f766e; } /* Verde oscuro para profes */
.teacher-btn:hover { background-color: #115e59; }
.feature-card.disabled { opacity: 0.7; box-shadow: none; }
.btn-disabled { background-color: #cbd5e1; color: #64748b; padding: 0.75rem 1.5rem; border: none; border-radius: 0.5rem; font-weight: bold; width: 100%; cursor: not-allowed; }
</style>