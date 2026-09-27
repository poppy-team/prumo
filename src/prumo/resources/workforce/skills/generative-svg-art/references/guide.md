# Reference: Algorithmic & Generative SVG Engineering

## 1. Mathematical Foundations of Generative Vector Art

### 1.1 Flow Fields (Vector Fields & Particle Advection)
A flow field is a 2D grid where each coordinate $(x, y)$ stores an angle $\theta(x, y)$ determining the direction of travel for virtual particles.
- **Noise Generator:** Evaluate 2D Perlin or Simplex noise at coordinate $(x \cdot s, y \cdot s)$ where $s$ is the frequency scale (typical $0.002 \le s \le 0.01$).
- **Angle Calculation:** $\theta(x, y) = \text{noise}(x \cdot s, y \cdot s) \times 2\pi \times \text{octaves}$.
- **Advection Integration:** A particle seeded at $(x_0, y_0)$ steps forward by step size $d$:
  $$x_{t+1} = x_t + d \cdot \cos(\theta(x_t, y_t))$$
  $$y_{t+1} = y_t + d \cdot \sin(\theta(x_t, y_t))$$
- **SVG Representation:** Group continuous particle trajectories into smooth cubic Bézier curves (`C` or `S` path commands) or polyline segments consolidated into a single `<path d="...">` element for maximal render performance.

### 1.2 Voronoi Diagrams & Delaunay Triangulation
- **Delaunay Condition:** For any circumcircle of a triangle in the network, no vertex lies in its interior.
- **Voronoi Dual:** The perpendicular bisectors of the Delaunay edges form polygon boundaries where each cell contains all points closer to its seed than to any other.
- **Lloyd's Relaxation:** Iteratively compute the centroid $C_i$ of each Voronoi polygon $P_i$ and move the seed point to $C_i$. After 3–5 iterations, the cells achieve uniform quasi-crystal distributions ideal for organic cell textures and mesh gradients.

### 1.3 L-Systems (Lindenmayer Systems) & Fractal Branching
Formal grammars defined as a tuple $G = (V, \omega, P)$:
- **Alphabet ($V$):** $\{F, +, -, [, ]\}$ where $F$ = move forward & draw, $+$ = turn right by $\delta$, $-$ = turn left by $\delta$, $[$ = push state, $]$ = pop state.
- **Axiom ($\omega$):** Starting string (e.g., $F$).
- **Production Rules ($P$):** e.g., $F \to FF+[+F-F-F]-[-F+F+F]$.
- **Execution:** Derive string to recursion depth $n$ ($n \le 5$ to respect DOM budgets) and translate state into SVG path commands.

---

## 2. Advanced SVG Filter Pipeline

SVG filters run on the GPU or raster pipeline of the browser. Proper structuring avoids CPU bottlenecks:

### 2.1 The Organic Topography / Marble Pipeline
Combines Perlin noise synthesis with spatial displacement:
```xml
<filter id="organic-displacement" x="-20%" y="-20%" width="140%" height="140%" filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB">
  <!-- 1. Generate Perlin Noise Base -->
  <feTurbulence 
    type="fractalNoise" 
    baseFrequency="0.015 0.03" 
    numOctaves="4" 
    seed="1337" 
    result="noise" />
  
  <!-- 2. Deform geometry using the noise red/green channels -->
  <feDisplacementMap 
    in="SourceGraphic" 
    in2="noise" 
    scale="36" 
    xChannelSelector="R" 
    yChannelSelector="G" 
    result="displaced" />
  
  <!-- 3. Enhance edge depth with diffuse lighting -->
  <feDiffuseLighting in="noise" surfaceScale="2" diffuseConstant="1.2" result="light">
    <feDistantLight azimuth="45" elevation="60" />
  </feDiffuseLighting>
  
  <!-- 4. Blend lighting and displaced graphic -->
  <feBlend in="displaced" in2="light" mode="multiply" result="blended" />
  <feComposite in="blended" in2="SourceGraphic" operator="in" />
</filter>
```

### 2.2 Glassmorphism & Caustic Shading Filter
```xml
<filter id="glass-specular" x="-10%" y="-10%" width="120%" height="120%">
  <feGaussianBlur in="SourceAlpha" stdDeviation="8" result="blur" />
  <feSpecularLighting in="blur" surfaceScale="5" specularConstant="1.5" specularExponent="30" result="specLight">
    <fePointLight x="100" y="-50" z="200" />
  </feSpecularLighting>
  <feComposite in="specLight" in2="SourceAlpha" operator="in" result="specOut" />
  <feBlend in="SourceGraphic" in2="specOut" mode="screen" />
</filter>
```

---

## 3. Deterministic Randomness & PRNG

True `Math.random()` breaks reproducibility and visual regression baselines. Always use a Seeded Pseudorandom Number Generator:

### Mulberry32 Implementation
```javascript
function mulberry32(seed) {
  return function() {
    let t = seed += 0x6D2B79F5;
    t = Math.imul(t ^ t >>> 15, t | 1);
    t ^= t + Math.imul(t ^ t >>> 7, t | 61);
    return ((t ^ t >>> 14) >>> 0) / 4294967296;
  }
}
```

---

## 4. Performance & DOM Budgets for SVG
- **Path Consolidation:** Instead of 1,000 `<line>` or `<path>` elements, consolidate all subpaths into a single `<path d="M... C... M... C...">`. This reduces DOM tree overhead by $\sim 95\%$.
- **Path Count Budget:** $\le 2,000$ curves per SVG for interactive web animations; $\le 8,000$ for static hero backgrounds.
- **Filter Regions:** Always define explicit margins on filters (`x="-20%" y="-20%" width="140%" height="140%"`) to eliminate clipping during displacement calculations.
- **Color Interpolation:** Explicitly set `color-interpolation-filters="sRGB"` to avoid dark banding and gamma shifts across Safari and Chromium.
