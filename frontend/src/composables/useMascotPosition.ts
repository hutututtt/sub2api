import { computed, nextTick, onMounted, ref, watch, type Ref } from 'vue'
import { useEventListener, useResizeObserver, useStorage } from '@vueuse/core'

type Position = { x: number; y: number }

export function useMascotPosition(
  root: Ref<HTMLElement | undefined>,
  trigger: Ref<HTMLButtonElement | undefined>,
  panel: Ref<HTMLElement | undefined>,
  onDrag: () => void
) {
  // Ratios preserve the chosen position across screen sizes and display modes.
  const saved = useStorage<Position>('sub2api:mascot:position:v1', { x: 1, y: 1 })
  const frame = ref({ left: 16, top: 16, right: window.innerWidth - 16, bottom: window.innerHeight - 16, width: 112, height: 162 })
  const panelSize = ref({ width: 360, height: 420 })
  const moving = ref<Position | null>(null)
  const dragging = ref(false)
  let gesture: { id: number; clientX: number; clientY: number; x: number; y: number } | null = null
  let suppressClick = false

  const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), Math.max(min, max))
  const ratio = (value: unknown) => typeof value === 'number' && Number.isFinite(value) ? clamp(value, 0, 1) : 1
  const position = computed(() => {
    const bounds = frame.value
    const maxX = Math.max(bounds.left, bounds.right - bounds.width)
    const maxY = Math.max(bounds.top, bounds.bottom - bounds.height)
    return {
      x: clamp(moving.value?.x ?? bounds.left + ratio(saved.value?.x) * (maxX - bounds.left), bounds.left, maxX),
      y: clamp(moving.value?.y ?? bounds.top + ratio(saved.value?.y) * (maxY - bounds.top), bounds.top, maxY)
    }
  })
  const rootStyle = computed(() => ({ left: `${position.value.x}px`, top: `${position.value.y}px`, right: 'auto', bottom: 'auto' }))
  const panelStyle = computed(() => {
    const bounds = frame.value
    const { x, y } = position.value
    const above = Math.max(0, y - bounds.top - 12)
    const below = Math.max(0, bounds.bottom - y - bounds.height - 12)
    const upward = above >= panelSize.value.height || above >= below
    const available = upward ? above : below
    return {
      left: `${clamp(x + bounds.width - panelSize.value.width, bounds.left, bounds.right - panelSize.value.width) - x}px`,
      top: `${upward ? -Math.min(panelSize.value.height, available) - 12 : bounds.height + 12}px`,
      bottom: 'auto',
      maxHeight: `${available}px`
    }
  })

  function measure() {
    if (!root.value) return
    const rect = root.value.getBoundingClientRect()
    const style = getComputedStyle(root.value)
    const inset = (side: string) => Math.max(window.innerWidth <= 640 ? 12 : 16, parseFloat(style.getPropertyValue(`--mascot-safe-${side}`)) || 0)
    const viewport = window.visualViewport
    const left = viewport?.offsetLeft ?? 0
    const top = viewport?.offsetTop ?? 0
    frame.value = {
      left: left + inset('left'), top: top + inset('top'),
      right: left + (viewport?.width ?? window.innerWidth) - inset('right'),
      bottom: top + (viewport?.height ?? window.innerHeight) - inset('bottom'),
      width: rect.width, height: rect.height
    }
    if (panel.value) {
      const panelRect = panel.value.getBoundingClientRect()
      panelSize.value = { width: panelRect.width, height: panelRect.height }
    }
  }

  function savePosition() {
    const bounds = frame.value
    saved.value = {
      x: (position.value.x - bounds.left) / Math.max(1, bounds.right - bounds.width - bounds.left),
      y: (position.value.y - bounds.top) / Math.max(1, bounds.bottom - bounds.height - bounds.top)
    }
  }

  function startDrag(event: PointerEvent) {
    if (gesture || event.button !== 0 || event.isPrimary === false) return
    measure()
    suppressClick = false
    gesture = { id: event.pointerId, clientX: event.clientX, clientY: event.clientY, ...position.value }
    trigger.value?.setPointerCapture?.(event.pointerId)
  }

  function moveDrag(event: PointerEvent) {
    if (!gesture || event.pointerId !== gesture.id) return
    const dx = event.clientX - gesture.clientX
    const dy = event.clientY - gesture.clientY
    if (!dragging.value && Math.hypot(dx, dy) < 6) return
    if (!dragging.value) onDrag()
    dragging.value = true
    suppressClick = true
    moving.value = { x: gesture.x + dx, y: gesture.y + dy }
    event.preventDefault()
  }

  function finishDrag(event?: PointerEvent) {
    if (!gesture || (event && event.pointerId !== gesture.id)) return
    const id = gesture.id
    gesture = null
    if (dragging.value) savePosition()
    moving.value = null
    dragging.value = false
    if (trigger.value?.hasPointerCapture?.(id)) trigger.value.releasePointerCapture(id)
  }

  function consumeDragClick(event: MouseEvent) {
    const consumed = suppressClick && event.detail !== 0
    suppressClick = false
    return consumed
  }

  function resetPosition() {
    finishDrag()
    saved.value = { x: 1, y: 1 }
  }

  useEventListener(window, 'pointermove', moveDrag, { passive: false })
  useEventListener(window, 'pointerup', finishDrag)
  useEventListener(window, 'pointercancel', finishDrag)
  useEventListener(trigger, 'lostpointercapture', finishDrag)
  useEventListener(window, 'blur', () => finishDrag())
  useEventListener(window, 'resize', measure)
  useEventListener(window.visualViewport, 'resize', measure)
  useEventListener(window.visualViewport, 'scroll', measure)
  useResizeObserver([root, panel], measure)
  watch([root, panel], () => { void nextTick(measure) })
  onMounted(measure)

  return { rootStyle, panelStyle, dragging, startDrag, consumeDragClick, resetPosition }
}
