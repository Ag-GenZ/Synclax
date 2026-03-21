import DefaultTheme from 'vitepress/theme'
import { h, defineComponent } from 'vue'
import { useData } from 'vitepress'
import GsapLanding from './GsapLanding.vue'

const Layout = defineComponent({
  name: 'AppLayout',
  setup() {
    const { frontmatter } = useData()
    return () => {
      if (frontmatter.value.layout === 'gsap-home') {
        return h(GsapLanding)
      }
      return h(DefaultTheme.Layout)
    }
  },
})

export default {
  extends: DefaultTheme,
  Layout,
}
