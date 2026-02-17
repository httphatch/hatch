<template>
  <div class="flow-diagram">
    <div class="flow-layout">
      <!-- Domain cards -->
      <div class="flow-col">
        <div
          class="flow-card flow-card--domain"
          v-for="route in routes"
          :key="route.domain"
        >
          <span class="flow-card__text">{{ route.domain }}</span>
          <div class="flow-card__lock" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
        </div>
      </div>

      <!-- Left curved connectors (domains → hub) -->
      <svg class="flow-svg" viewBox="0 0 100 100" preserveAspectRatio="none">
        <path
          v-for="(p, i) in leftPaths"
          :key="'lp'+i"
          :d="p"
          fill="none"
          stroke="#e2d6c8"
          stroke-width="2"
          vector-effect="non-scaling-stroke"
        />
        <template v-for="(p, i) in leftPaths" :key="'ld'+i">
          <circle r="2.5" fill="#f28482" class="flow-dot" vector-effect="non-scaling-stroke">
            <animateMotion :dur="dur" repeatCount="indefinite" :path="p" :begin="`${i * 0.7}s`" />
          </circle>
          <circle r="2.5" fill="#f28482" class="flow-dot" vector-effect="non-scaling-stroke">
            <animateMotion :dur="dur" repeatCount="indefinite" :path="p" :begin="`${i * 0.7 + 1}s`" />
          </circle>
        </template>
      </svg>

      <!-- Center hub -->
      <div class="flow-hub">
        <span class="flow-hub__label">Hatch</span>
      </div>

      <!-- Right curved connectors (hub → ports) -->
      <svg class="flow-svg" viewBox="0 0 100 100" preserveAspectRatio="none">
        <path
          v-for="(p, i) in rightPaths"
          :key="'rp'+i"
          :d="p"
          fill="none"
          stroke="#e2d6c8"
          stroke-width="2"
          vector-effect="non-scaling-stroke"
        />
        <template v-for="(p, i) in rightPaths" :key="'rd'+i">
          <circle r="2.5" fill="#f28482" class="flow-dot" vector-effect="non-scaling-stroke">
            <animateMotion :dur="dur" repeatCount="indefinite" :path="p" :begin="`${i * 0.7 + 0.3}s`" />
          </circle>
          <circle r="2.5" fill="#f28482" class="flow-dot" vector-effect="non-scaling-stroke">
            <animateMotion :dur="dur" repeatCount="indefinite" :path="p" :begin="`${i * 0.7 + 1.3}s`" />
          </circle>
        </template>
      </svg>

      <!-- Port cards -->
      <div class="flow-col">
        <div
          class="flow-card flow-card--port"
          v-for="route in routes"
          :key="route.port"
        >
          <span class="flow-card__text">{{ route.port }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const routes: { domain: string; port: string }[] = [
  { domain: 'myapp.test', port: 'localhost:3000' },
  { domain: 'api.myapp.test', port: 'localhost:8080' },
  { domain: 'admin.myapp.test', port: 'localhost:4000' },
]

const dur = '2s'

// Card centers at ~13.5%, 50%, ~86.5% of column height
// (3 cards × ~40px + 2 gaps × 14px = 148px total)
const y1 = 13.5
const y2 = 50
const y3 = 86.5

const leftPaths = [
  `M 0 ${y1} C 60 ${y1}, 40 ${y2}, 100 ${y2}`,
  `M 0 ${y2} L 100 ${y2}`,
  `M 0 ${y3} C 60 ${y3}, 40 ${y2}, 100 ${y2}`,
]

const rightPaths = [
  `M 0 ${y2} C 60 ${y2}, 40 ${y1}, 100 ${y1}`,
  `M 0 ${y2} L 100 ${y2}`,
  `M 0 ${y2} C 60 ${y2}, 40 ${y3}, 100 ${y3}`,
]
</script>

<style scoped>
.flow-diagram {
  max-width: 860px;
  margin: 0 auto;
  padding: 16px 24px 48px;
}

/* ── Layout ── */

.flow-layout {
  display: flex;
  align-items: stretch;
  gap: 0;
}

.flow-col {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 14px;
  flex-shrink: 0;
}

.flow-svg {
  flex: 1;
  min-width: 40px;
  overflow: visible;
}

/* ── Cards ── */

.flow-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 8px;
  background: #ffffff;
  border: 1.5px solid #e2d6c8;
  white-space: nowrap;
  justify-content: center;
}

.flow-card--domain {
  position: relative;
  padding-right: 28px;
}

.flow-card__text {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.88rem;
  font-weight: 600;
  color: #3d3d3d;
}

.flow-card--port .flow-card__text {
  font-weight: 500;
  color: #6b6b6b;
}

/* ── Lock circle on card edge ── */

.flow-card__lock {
  position: absolute;
  right: -14px;
  top: 50%;
  transform: translateY(-50%);
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(
    90deg,
    #ffffff 0%,
    #ffffff 50%,
    #e2d6c8 50%,
    #e2d6c8 100%
  );
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
}

.flow-card__lock::before {
  content: '';
  position: absolute;
  inset: 1.5px;
  border-radius: 50%;
  background: #ffffff;
}

.flow-card__lock svg {
  position: relative;
  z-index: 1;
  width: 13px;
  height: 13px;
  color: #84a59d;
}

/* ── Center hub ── */

.flow-hub {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  z-index: 1;
}

.flow-hub__label {
  padding: 10px 22px;
  border-radius: 24px;
  background: #84a59d;
  color: #ffffff;
  font-family: 'Fugaz One', cursive;
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  box-shadow: 0 2px 12px rgba(132, 165, 157, 0.3);
}

/* ── Animated dots ── */

.flow-dot {
  opacity: 0.85;
}

/* ── Reduced motion ── */

@media (prefers-reduced-motion: reduce) {
  .flow-dot {
    display: none;
  }
}

/* ── Mobile ── */

@media (max-width: 680px) {
  .flow-diagram {
    padding: 12px 12px 32px;
  }

  .flow-col {
    gap: 10px;
  }

  .flow-card {
    padding: 8px 10px;
  }

  .flow-card--domain {
    padding-right: 22px;
  }

  .flow-card__text {
    font-size: 0.78rem;
  }

  .flow-card__lock {
    width: 24px;
    height: 24px;
    right: -12px;
  }

  .flow-card__lock svg {
    width: 11px;
    height: 11px;
  }

  .flow-hub__label {
    padding: 8px 14px;
    font-size: 0.78rem;
  }

  .flow-svg {
    min-width: 20px;
  }
}

@media (max-width: 480px) {
  .flow-card__text {
    font-size: 0.72rem;
  }

  .flow-hub__label {
    padding: 6px 10px;
    font-size: 0.72rem;
  }
}
</style>
