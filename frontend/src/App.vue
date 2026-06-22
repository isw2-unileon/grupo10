<script setup lang="ts">
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function onLogout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <header>
    <nav>
      <RouterLink v-if="auth.user?.role === 'student'" to="/student">
        Mi Portal (Estudiante)
      </RouterLink>
      <RouterLink v-else-if="auth.user?.role === 'teacher'" to="/teacher">
        Mi Portal (Profesor)
      </RouterLink>
      <RouterLink v-else to="/">
        Inicio
      </RouterLink>
      
      <template v-if="auth.user">
        <button type="button" class="logout" @click="onLogout">
          Cerrar sesión
        </button>
      </template>

      <template v-else>
        <RouterLink to="/login">Iniciar sesión</RouterLink>
        <RouterLink to="/register">Registrarse</RouterLink>
      </template>
    </nav>

    <div class="user-status" v-if="auth.user">
      Sesión iniciada como: <strong>{{ auth.user.email }}</strong>
      (Rol: <em>{{ auth.user.role }}</em>)
    </div>
  </header>

  <main>
    <RouterView />
  </main>
</template>

<style scoped>
header {
  border-bottom: 1px solid #e2e8f0;
  margin-bottom: 1.5rem;
}

nav {
  display: flex;
  gap: 1.5rem;
  padding: 1rem;
  align-items: center;
}

nav a {
  text-decoration: none;
  color: #334155;
}

nav a.router-link-active {
  font-weight: 600;
  color: #2563eb; 
}

.logout {
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  color: #dc2626; 
  cursor: pointer;
  text-decoration: underline;
  margin-left: auto; /* Esto empuja el botón al final derecho de la barra */
}

.user-status {
  background-color: #f8fafc;
  padding: 0.5rem 1rem;
  font-size: 0.85rem;
  color: #64748b;
  text-align: right;
}
</style>
