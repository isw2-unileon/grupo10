<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <header>
    <nav>
      <RouterLink to="/">Inicio</RouterLink>
      <RouterLink to="/login">Iniciar sesión</RouterLink>
      <RouterLink to="/register">Registrarse</RouterLink>
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
</style>
