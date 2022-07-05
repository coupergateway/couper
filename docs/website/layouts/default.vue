<script setup lang="ts">
import Navbar from "~/components/Navbar.vue";
import SideNavbar from "~/components/SideNavbar.vue";

const { toc } = useContent();
</script>

<template>
  <div class="justify-between relative">
    <div class="grid grid-cols-1 gap-x-4 bg-slate-50">
      <Navbar />
    </div>
    <div class="flex">
      <SideNavbar class="overflow-y-scroll grow w-10 bg-slate-700 p-4 text-gray-400" />
      <main class="overflow-y-scroll grow max-w-7xl p-4 sm:px-6
        prose prose-slate
        prose-code:bg-sky-100 prose-code:text-sky-900 prose-code:p-1 prose-code:rounded-md prose-code:text-sm
        prose-a:text-sky-600
        hover:prose-a:text-amber-500
      ">
        <slot />
      </main>
      <nav v-if="toc && toc.links" class="grow w-10 bg-slate-700 p-4 text-gray-400">
          <h2 class="text-gray-100">On this Page</h2>
          <ul v-if="toc && toc.links">
            <li v-for="link in toc.links" :key="link.text">
              <a :href="`#${link.id}`">
                {{ link.text }}
              </a>
            </li>
          </ul>
      </nav>
    </div>
  </div>
</template>

<style>
pre > code {
  /* since code bg is bright, disable within pre */
  background-color: inherit !important;
  padding: 0 !important;
}
</style>
