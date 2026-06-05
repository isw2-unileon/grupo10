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
      <RouterLink to="/">Inicio</RouterLink>
      <template v-if="auth.isAuthenticated">
        <button type="button" class="logout" @click="onLogout">
          Cerrar sesión
        </button>
      </template>
      <template v-else>
        <RouterLink to="/login">Iniciar sesión</RouterLink>
        <RouterLink to="/register">Registrarse</RouterLink>
      </template>
    </nav>
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
  gap: 1rem;
  padding: 1rem;
}

nav a.router-link-active {
  font-weight: 600;
}

.logout {
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  color: inherit;
  cursor: pointer;
  text-decoration: underline;
}
</style>
