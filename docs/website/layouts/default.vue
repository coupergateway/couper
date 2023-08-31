<script setup lang="ts">
import Logo from "@/components/content/Logo";
import GithubIcon from "@/components/content/GithubIcon";
import SearchBar from "@/components/SearchBar";
import SideNavbar from "@/components/SideNavbar";
</script>

<template>
  <div class="antialiasing text-slate-700">
    <header class="flex justify-between p-1 headerShadow">
      <div class="relative w-2/5 flex items-center m-4">
        <Logo class="box-border h-16 w-16 md:w-18 md:h-18 xl:w-22 xl:h-22" />
        <h1 class="text-2xl font-medium align-top p-4">Couper Documentation</h1>
        <div class="box-border text-base">
          <div class="rounded-md bg-lime-500 p-1">edge</div>
        </div>
      </div>
      <div class="flex justify-around w-full">
        <SearchBar class="px-4 sm:px-8 lg:px-16 xl:px-5 mx-auto" />
      </div>
      <div class="w-1/5"><a href="https://github.com/coupergateway/couper" target="_blank"><github-icon class="w-8 h-8 mt-8" /></a></div>
    </header>
    <div class="flex justify-between m-2 p-2 min-h-screen">
      <SideNavbar class="w-1/5 min-w-min grow-0 p-2" />
      <main class="grow p-5 w-3/5
        prose prose-slate prose-lg prose-code:bg-sky-100 prose-code:text-sky-900 prose-code:p-1 prose-code:rounded-md prose-code:text-sm prose-blockquote:bg-purple-100 prose-blockquote:rounded-md prose-a:text-sky-600 hover:prose-a:text-amber-500">
        <slot />
      </main>
      <PageToc/>
    </div>
    <footer class="col-span-3 text-sm leading-6 m-10"><p>Copyright couper.io</p></footer>
  </div>
</template>

<style>
pre > code {
  /* since code bg is bright, disable within pre */
  background-color: inherit !important;
  padding: 0 !important;
  white-space: pre !important /* line breaks for copy/paste! */
}

.headerShadow {
  box-shadow: rgb(203 203 203) 0 5px 10px 0;
}
</style>
