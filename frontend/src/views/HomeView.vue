<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <section class="home">
    <h1>Learning Platform</h1>
    <p>
      Plataforma integral para optimizar la interacción alumno–profesor
      mediante IA, con feedback instantáneo sobre los apuntes.
    </p>
    <p v-if="auth.isAuthenticated">
      Sesión iniciada como <strong>{{ auth.user?.email }}</strong>
    </p>
    <p v-else>No has iniciado sesión.</p>
  </section>
</template>

<style scoped>
.home {
  max-width: 640px;
  margin: 0 auto;
  padding: 0 1rem;
}
</style>
