<template>
    <CThemeProvider>
      <CReset />
      <CColorModeProvider>
        <MDXProvider :components="Mapping">
          <readme/>
        </MDXProvider>
      </CColorModeProvider>
    </CThemeProvider>
</template>

<script>
import { MDXProvider } from "@mdx-js/vue"
import MDXComponents from '~/components/MDXComponents'

import {
  CThemeProvider,
  CColorModeProvider,
  CReset
} from '@chakra-ui/vue'

export default {
  name: 'IndexPage',
  components: {
    MDXProvider,
    MDXComponents,
    // UI
    CThemeProvider,
    CColorModeProvider,
    CReset,
    // Content
  },
  data: () => ({
    MDXComponents
  })
}
</script>

<style>
.container {
  margin: 0 auto;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
}

.title {
  font-family:
    Quicksand,
    "Source Sans Pro",
    -apple-system,
    BlinkMacSystemFont,
    "Segoe UI",
    Roboto,
    "Helvetica Neue",
    Arial,
    sans-serif;
  display: block;
  font-weight: 300;
  font-size: 100px;
  color: #35495e;
  letter-spacing: 1px;
}

.subtitle {
  font-weight: 300;
  font-size: 42px;
  color: #526488;
  word-spacing: 5px;
  padding-bottom: 15px;
}

.links {
  padding-top: 15px;
}
</style>
