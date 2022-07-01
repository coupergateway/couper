<script setup lang="ts">
import Navbar from "~/components/Navbar.vue";
import SideNavbar from "~/components/SideNavbar.vue";
</script>

<template>
  <div class="flex min-h-screen w-full flex-row0">
    <div>
      <Navbar class="mx-auto max-w-full px-4 py-4 overflow-x-auto" />
      <div class="flex">
        <SideNavbar class="flex-initial overscroll-y-auto px-4 py-4" />
        <div class="flex-auto prose p-4 sm:px-6 mx-auto max-w-full lg:max-w-7xl h-full pb-6">
          <main>
            <slot />
          </main>
        </div>
      </div>
    </div>
<!--    <Footer />-->
  </div>
</template>
