import { definePreset } from '@primeuix/themes'
import Aura from '@primeuix/themes/aura'

// A focused blue, matched to the design system chosen for ztpanel:
// trustworthy/infra-tool blue as primary, neutral slate grays for
// surfaces and text. Kept close to Aura's defaults so PrimeVue
// components still look coherent out of the box.
export const ztpanelTheme = definePreset(Aura, {
  semantic: {
    primary: {
      50: '#eff6ff',
      100: '#dbeafe',
      200: '#bfdbfe',
      300: '#93c5fd',
      400: '#60a5fa',
      500: '#2563eb',
      600: '#1d4ed8',
      700: '#1e40af',
      800: '#1e3a8a',
      900: '#172554',
      950: '#0f172a',
    },
  },
})
