<template>
  <div>
    <h4>Login</h4>
    <form @submit.prevent="doLogin" class="mt-3" style="max-width:400px">
      <div class="mb-3">
        <label class="form-label">Username</label>
        <input v-model="userField" class="form-control" required />
      </div>
      <div class="mb-3">
        <label class="form-label">Password</label>
        <input v-model="passField" type="password" class="form-control" required />
      </div>
      <button class="btn btn-primary">Login</button>
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
