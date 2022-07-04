<script setup lang="ts">
import Navbar from "~/components/Navbar.vue";
import SideNavbar from "~/components/SideNavbar.vue";

const { toc } = useContent();
</script>

<template>
  <div class="grid grid-cols-1 gap-x-4 bg-slate-50">
    <Navbar />
    <div class="flex">
      <SideNavbar class="flex-none w-100 bg-slate-700 p-4 text-gray-400" />
      <main class="grow max-w-full prose p-4 sm:px-6">
        <slot />
      </main>
      <div class="flex-none w-100 bg-slate-700 p-4 text-gray-400">
        <nav>
          <div>
            <ul v-if="toc && toc.links">
              <li v-for="link in toc.links" :key="link.text">
                <a :href="`#${link.id}`">
                  {{ link.text }}
                </a>
              </li>
            </ul>
          </div>
        </nav>
      </div>
    </div>
<!--    <Footer />-->
  </div>
</template>
