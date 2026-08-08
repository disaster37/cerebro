<template>
  <nav class="navbar navbar-expand-lg navbar-dark bg-dark fixed-top">
    <div class="container-fluid">
      <a class="navbar-brand" href="/">cerebro</a>
      <div class="collapse navbar-collapse">
        <ul class="navbar-nav me-auto">
          <li class="nav-item"><NuxtLink class="nav-link" to="/connect">{{ clusterName || 'Connect' }}<BackendBadge :backend-type="backendType" :backend-version="backendVersion" /></NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/overview">Overview</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/nodes">Nodes</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/rest">REST</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/aliases">Aliases</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/create">Create Index</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/analysis">Analysis</NuxtLink></li>
          <li class="nav-item" v-if="clusterName"><NuxtLink class="nav-link" to="/templates">Templates</NuxtLink></li>
          <li class="nav-item dropdown" v-if="clusterName">
            <a class="nav-link dropdown-toggle" href="#" data-bs-toggle="dropdown">More</a>
            <ul class="dropdown-menu">
              <li><NuxtLink class="dropdown-item" to="/cluster-settings">Cluster Settings</NuxtLink></li>
              <li><NuxtLink class="dropdown-item" to="/index-settings">Index Settings</NuxtLink></li>
              <li><NuxtLink class="dropdown-item" to="/snapshot">Snapshot</NuxtLink></li>
              <li><NuxtLink class="dropdown-item" to="/repository">Repositories</NuxtLink></li>
              <li><NuxtLink class="dropdown-item" to="/cat">CAT</NuxtLink></li>
            </ul>
          </li>
        </ul>
        <div class="navbar-text me-2" v-if="user">{{ user }}</div>
        <a v-if="!user" class="nav-link text-light" href="/login">Login</a>
        <a v-if="user" class="nav-link text-light" href="javascript:void(0)" @click="logout">Logout</a>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName', () => '')
const user = useState<string>('user', () => '')
const backendType = useState<string>('backendType', () => '')
const backendVersion = useState<string>('backendVersion', () => '')

async function logout() {
  await $fetch('/auth/logout', { method: 'POST' })
  navigateTo('/login')
}
</script>
