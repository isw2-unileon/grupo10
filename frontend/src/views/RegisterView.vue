<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import { useAuthStore, type Role } from '@/stores/auth'
import type { ApiError } from '@/services/api'

const auth = useAuthStore()
const router = useRouter()

const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const role = ref<Role>('student')
const error = ref<string | null>(null)
const loading = ref(false)

async function onSubmit() {
  error.value = null

  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  loading.value = true
  try {
    await auth.register({
      name: name.value,
      email: email.value,
      password: password.value,
      role: role.value,
    })
    // register() already stores the session, so go straight to Home.
    router.push({ name: 'home' })
  } catch (err) {
    error.value = (err as ApiError).message || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="register">
    <h1>Create account</h1>
    <form @submit.prevent="onSubmit">
      <label>
        Name
        <input v-model="name" type="text" required autocomplete="name" />
      </label>
      <label>
        Email
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label>
        I am a
        <select v-model="role" required>
          <option value="student">Student</option>
          <option value="teacher">Teacher</option>
        </select>
      </label>
      <label>
        Password
        <input
          v-model="password"
          type="password"
          required
          minlength="6"
          autocomplete="new-password"
        />
      </label>
      <label>
        Confirm password
        <input
          v-model="confirmPassword"
          type="password"
          required
          minlength="6"
          autocomplete="new-password"
        />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">
        {{ loading ? 'Creating account…' : 'Create account' }}
      </button>
    </form>
    <p class="alt">
      Already have an account?
      <RouterLink to="/login">Sign in</RouterLink>
    </p>
  </section>
</template>

<style scoped>
.register {
  max-width: 360px;
  margin: 0 auto;
  padding: 0 1rem;
}

form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

select {
  padding: 0.5rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.375rem;
}

.error {
  color: #dc2626;
}

.alt {
  margin-top: 1rem;
  font-size: 0.9rem;
}
</style>
