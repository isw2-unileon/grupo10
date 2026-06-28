<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

// Redirect logic based on user role
const redirectUser = () => {
  if (auth.isAuthenticated && auth.user) {
    if (auth.user.role === 'student') {
      router.push('/student/groups')
    } else if (auth.user.role === 'teacher') {
      // Assuming your teacher route is /teacher/groups. Change if needed!
      router.push('/teacher/groups') 
    }
  }
}

onMounted(() => {
  redirectUser()
})

// Watch for authentication state changes (e.g., just logged in)
watch(() => auth.isAuthenticated, () => {
  redirectUser()
})
</script>

<template>
  <section class="home">
    <h1>Learning Platform</h1>
    <p>
      Plataforma integral para optimizar la interacción alumno–profesor
      mediante IA, con feedback instantáneo sobre los apuntes.
    </p>
    <p v-if="auth.isAuthenticated">
      Sesión iniciada como <strong>{{ auth.user?.email }}</strong>. Redirigiendo...
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
