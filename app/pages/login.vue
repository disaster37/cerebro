<template>
  <div>
    <h4>Login</h4>
    <form @submit.prevent="doLogin" class="mt-3 max-w-sm">
      <div class="mb-3">
        <UFormField label="Username">
          <UInput v-model="userField" required class="w-full" />
        </UFormField>
      </div>
      <div class="mb-3">
        <UFormField label="Password">
          <UInput v-model="passField" type="password" required class="w-full" />
        </UFormField>
      </div>
      <UButton type="submit">Login</UButton>
    </form>
  </div>
</template>

<script setup lang="ts">
const { login } = useAuth()
const userField = ref('')
const passField = ref('')

async function doLogin() {
  const ok = await login(userField.value, passField.value)
  if (ok) navigateTo('/')
}
</script>
