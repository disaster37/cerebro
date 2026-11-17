import { computed } from 'vue'
import {
  ArcElement,
  BarElement,
  CategoryScale,
  Chart,
  Legend,
  LinearScale,
  Tooltip,
} from 'chart.js'

Chart.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

export interface ChartThemeTokens {
  isDark: boolean
  cardClass: string
  textColor: string
  mutedTextColor: string
  gridColor: string
  borderColor: string
  tooltipBg: string
  tooltipBorder: string
  series: {
    healthy: string
    replica: string
    warning: string
    relocating: string
    danger: string
    track: string
  }
}

export function useChartTheme() {
  const colorMode = useColorMode()
  const isDark = computed(() => colorMode.value === 'dark')

  const tokens = computed<ChartThemeTokens>(() => {
    if (isDark.value) {
      return {
        isDark: true,
        cardClass: 'bg-neutral-950 ring-1 ring-neutral-800',
        textColor: '#d4d4d8',
        mutedTextColor: '#a1a1aa',
        gridColor: '#262626',
        borderColor: '#09090b',
        tooltipBg: '#000000',
        tooltipBorder: '#3f3f46',
        series: {
          healthy: '#10b981',
          replica: '#38bdf8',
          warning: '#f59e0b',
          relocating: '#a78bfa',
          danger: '#f43f5e',
          track: '#1f2937',
        },
      }
    }
    return {
      isDark: false,
      cardClass: 'bg-white ring-1 ring-neutral-200',
      textColor: '#374151',
      mutedTextColor: '#6b7280',
      gridColor: '#e5e7eb',
      borderColor: '#ffffff',
      tooltipBg: '#ffffff',
      tooltipBorder: '#e5e7eb',
      series: {
        healthy: '#10b981',
        replica: '#38bdf8',
        warning: '#f59e0b',
        relocating: '#a78bfa',
        danger: '#f43f5e',
        track: '#e5e7eb',
      },
    }
  })

  const doughnutOptions = computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    cutout: '68%',
    plugins: {
      legend: {
        position: 'bottom' as const,
        labels: { color: tokens.value.textColor, boxWidth: 10, boxHeight: 10 },
      },
      tooltip: {
        backgroundColor: tokens.value.tooltipBg,
        borderColor: tokens.value.tooltipBorder,
        borderWidth: 1,
        titleColor: tokens.value.textColor,
        bodyColor: tokens.value.textColor,
      },
    },
  }))

  const barOptions = computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    indexAxis: 'y' as const,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: tokens.value.tooltipBg,
        borderColor: tokens.value.tooltipBorder,
        borderWidth: 1,
        titleColor: tokens.value.textColor,
        bodyColor: tokens.value.textColor,
      },
    },
    scales: {
      x: {
        min: 0,
        grid: { color: tokens.value.gridColor },
        ticks: { color: tokens.value.mutedTextColor },
      },
      y: {
        grid: { color: tokens.value.gridColor },
        ticks: { color: tokens.value.textColor },
      },
    },
  }))

  return { isDark, tokens, doughnutOptions, barOptions }
}
