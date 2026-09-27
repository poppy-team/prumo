# Generative SVG Artwork Specification Template

## 1. Metadata & Algorithm Profile

- **Artwork ID:** `[kebab-case-name]`
- **Algorithm Type:** `[flow-field | voronoi-lloyd | l-system | perlin-contour | reaction-diffusion]`
- **Deterministic Seed:** `[integer / string]`
- **Aspect Ratio & Canvas:** `viewBox="0 0 {WIDTH} {HEIGHT}"`
- **Target Use Case:** `[hero-background | generative-avatar | data-visualization-texture | abstract-card]`

---

## 2. Parameter Schema

```json
{
  "seed": 42071,
  "dimensions": { "width": 1200, "height": 800 },
  "algorithm": {
    "type": "flow-field",
    "step_size": 4.5,
    "steps_per_particle": 85,
    "particle_count": 450,
    "noise_frequency": 0.004,
    "octaves": 3
  },
  "palette": {
    "background": "var(--vp-c-bg-alt, #0d1322)",
    "primary_gradient": ["oklch(0.65 0.22 250)", "oklch(0.75 0.18 190)"],
    "accent": "oklch(0.85 0.15 85)"
  },
  "filters": {
    "turbulence_freq": "0.012 0.02",
    "displacement_scale": 24
  }
}
```

---

## 3. SVG Assembly Boilerplate

```xml
<svg 
  xmlns="http://www.w3.org/2000/svg" 
  viewBox="0 0 1200 800" 
  width="100%" 
  height="100%" 
  role="img" 
  aria-labelledby="title-art desc-art"
  class="generative-svg-canvas">
  
  <title id="title-art">[Title of Procedural Artwork]</title>
  <desc id="desc-art">[Description of the algorithmic landscape and colors generated]</desc>

  <defs>
    <!-- 1. Color Palettes and Gradients -->
    <linearGradient id="flow-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="var(--gen-color-1, #3b82f6)" />
      <stop offset="100%" stop-color="var(--gen-color-2, #06b6d4)" />
    </linearGradient>

    <!-- 2. Procedural Filter Pipeline -->
    <filter id="procedural-warp" x="-20%" y="-20%" width="140%" height="140%" color-interpolation-filters="sRGB">
      <feTurbulence type="fractalNoise" baseFrequency="0.008" numOctaves="3" seed="42" result="noise" />
      <feDisplacementMap in="SourceGraphic" in2="noise" scale="18" xChannelSelector="R" yChannelSelector="G" />
    </filter>
  </defs>

  <!-- Background Base -->
  <rect width="100%" height="100%" fill="var(--gen-bg, #090d16)" />

  <!-- Consolidated Generative Path Geometry -->
  <g filter="url(#procedural-warp)" opacity="0.85">
    <path 
      d="[CONSOLIDATED_CURVE_DATA]" 
      fill="none" 
      stroke="url(#flow-gradient)" 
      stroke-width="1.5" 
      stroke-linecap="round" />
  </g>
</svg>
```
