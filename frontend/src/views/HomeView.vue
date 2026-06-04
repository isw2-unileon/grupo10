<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <section class="home">
    <h1>Learning Platform</h1>
    <p>
      Integrated platform to optimize student–teacher interaction using AI for
      instant feedback on notes.
    </p>
    <p v-if="auth.isAuthenticated">
      Signed in as <strong>{{ auth.user?.email }}</strong>
    </p>
    <p v-else>You are not signed in.</p>
  </section>
</template>

<style scoped>
.home {
  max-width: 640px;
  margin: 0 auto;
  padding: 0 1rem;
}
</style>
