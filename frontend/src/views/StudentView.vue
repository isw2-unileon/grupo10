<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { RouterLink } from 'vue-router'

const auth = useAuthStore()

// Para mostrar solo el nombre antes del @ en el email
const userName = auth.user?.email.split('@')[0] || 'Estudiante'
</script>

<template>
  <div class="dashboard-container">
    <header class="welcome-section">
      <h1>¡Hola, {{ userName }}! 👋</h1>
      <p>Bienvenido a tu panel de control. ¿Qué vamos a aprender hoy?</p>
    </header>

    <div class="tools-grid">
      <article class="tool-card">
        <div class="tool-icon">📝</div>
        <div class="tool-info">
          <h2>Mis Apuntes e IA</h2>
          <p>Redacta tus notas de clase, recibe feedback instantáneo de la Inteligencia Artificial y envíalos a tu profesor para que los apruebe.</p>
          <RouterLink to="/student/notes" class="btn-primary">
            Ir al Editor de Apuntes
          </RouterLink>
        </div>
      </article>

      <article class="tool-card ">
        <div class="tool-icon">📅</div>
        <div class="tool-info">
          <h2>Tutorías (Próximamente)</h2>
          <p>Reserva horarios con tus profesores para resolver dudas específicas de la asignatura.</p>
          <RouterLink to="/calendar" class="btn-action teacher-btn">Gestionar Calendario</RouterLink>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
.dashboard-container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 2rem 1rem;
}

.welcome-section {
  margin-bottom: 2.5rem;
}

.welcome-section h1 {
  color: #1e293b;
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.welcome-section p {
  color: #64748b;
  font-size: 1.1rem;
}

.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
}

.tool-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 0.75rem;
  padding: 1.5rem;
  display: flex;
  gap: 1.5rem;
  box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.05);
  transition: transform 0.2s ease;
}

.tool-card:hover:not(.disabled) {
  transform: translateY(-4px);
}

.tool-card.disabled {
  opacity: 0.6;
  box-shadow: none;
}

.tool-icon {
  font-size: 3rem;
}

.tool-info {
  display: flex;
  flex-direction: column;
}

.tool-info h2 {
  color: #334155;
  margin-bottom: 0.5rem;
  font-size: 1.25rem;
}

.tool-info p {
  color: #475569;
  font-size: 0.95rem;
  margin-bottom: 1.5rem;
  flex-grow: 1;
}

.btn-primary {
  background-color: #2563eb;
  color: white;
  text-decoration: none;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  text-align: center;
  font-weight: 600;
  width: 100%;
}

.btn-primary:hover {
  background-color: #1d4ed8;
}

.btn-disabled {
  background-color: #cbd5e1;
  color: #64748b;
  padding: 0.75rem 1rem;
  border: none;
  border-radius: 0.5rem;
  font-weight: 600;
  cursor: not-allowed;
  width: 100%;
}
</style>