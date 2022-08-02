<script setup lang="ts">
import Navbar from "~/components/Navbar.vue";
import SideNavbar from "~/components/SideNavbar.vue";

const { toc } = useContent();
</script>

<template>
  <div class="antialiasing text-slate-500 bg-white">
    <div class="absolute z-20 top-0 inset-x-0 flex justify-center overflow-hidden pointer-events-none">
    </div>
    <Navbar class="sticky top-0 z-40 w-full backdrop-blur flex-none transition-colors duration-500 lg:z-50 lg:border-b lg:border-slate-900/10 dark:border-slate-50/[0.06] bg-white supports-backdrop-blur:bg-white/95 dark:bg-slate-700/90" />
    <div class="hidden lg:block fixed z-20 inset-0 pt-[7.56rem] left-0 right-auto w-[15rem] overflow-y-auto">
      <SideNavbar class="lg:text-sm lg:leading-6
      bg-slate-600 text-gray-400 pl-4 pt-2" />
    </div>
    <div class="max-w-8xl mx-auto sm:px-6">
      <div class="lg:pl-[17rem]">
          <main class="max-w-8xl pt-6 xl:max-w-none xl:ml-0 xl:mr-[15.5rem] xl:pr-16 relative z-100 mt-4
            prose prose-slate
            prose-code:bg-sky-100 prose-code:text-sky-900 prose-code:p-1 prose-code:rounded-md prose-code:text-sm
            prose-blockquote:bg-purple-100
            prose-blockquote:rounded-md
            prose-a:text-sky-600
            hover:prose-a:text-amber-500
          ">
            <slot />
          </main>
        <footer class="text-sm leading-6 mt-12"><p>Copyright couper.io</p></footer>
      </div>
        <nav v-if="toc && toc.links" class="fixed z-20 top-[7.56rem] bottom-0 right-0 w-[19.5rem] py-8 px-6 overflow-y-auto hidden xl:block
        bg-slate-600 text-gray-400">
            <h2 class="text-gray-100">On this Page</h2>
            <ul v-if="toc && toc.links">
              <li v-for="link in toc.links" :key="link.text">
                <NuxtLink :href="`#${link.id}`">
                  {{ link.text }}
                </NuxtLink>
                <ul v-if="link.children">
                  <li v-for="child in link.children" :key="child.id">
                    <NuxtLink class="pl-2" :href="`#${child.id}`">
                      {{ child.text }}
                    </NuxtLink>
                  </li>
                </ul>
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
  white-space: pre !important /* line breaks for copy/paste! */
}
</style>
