<template>
  <div>
    <h4>Login</h4>
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      :description="error"
      role="alert"
      data-testid="login-error"
      class="mt-3 max-w-sm"
    />
    <form class="mt-3 max-w-sm" @submit.prevent="doLogin">
      <div class="mb-3">
        <UFormField label="Username">
          <UInput v-model="userField" required class="w-full" data-testid="login-username" />
        </UFormField>
      </div>
      <div class="mb-3">
        <UFormField label="Password">
          <UInput v-model="passField" type="password" required class="w-full" data-testid="login-password" />
        </UFormField>
      </div>
      <UButton type="submit" data-testid="login-submit">Login</UButton>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { consumeRedirectCookie, useAuth } from '../composables/useAuth'

const { login } = useAuth()
const userField = ref('')
const passField = ref('')
const error = ref('')

async function doLogin() {
  error.value = ''
  const ok = await login(userField.value, passField.value)
  if (ok) {
    navigateTo(consumeRedirectCookie())
  } else {
    error.value = 'Invalid username or password'
  }
}
</script>
