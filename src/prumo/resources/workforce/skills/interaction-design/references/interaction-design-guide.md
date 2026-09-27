# Interaction Design (IxD) & Motion Physics: Technical Reference Guide

## 1. Ergonomic Laws of Interaction

### 1.1 Fitts's Law
Formulated by Paul Fitts in 1954, it predicts the time $T$ required to rapidly move to a target area:
$$T = a + b \log_2\left(1 + \frac{D}{W}\right)$$
- $D$: Distance to the target center.
- $W$: Width of the target along the dimension of motion.
- $\log_2(1 + D/W)$: The **Index of Difficulty** ($ID$), measured in bits.

#### Key Engineering Takeaways
1. **Screen Edges & Corners**: On desktop, the screen edges have infinite effective width ($W = \infty$) because the mouse cursor cannot overshoot the physical display boundary. Pin high-frequency menus to screen edges.
2. **Hit Target Inflation**: Use invisible padding around visual buttons (`padding: 12px` inside an absolute hit area) to expand $W$ without altering visual aesthetics.

### 1.2 Hick-Hyman Law
$$T = b \cdot \log_2(n + 1)$$
- Reaction time increases logarithmically with the number of alternatives $n$. Keep decision menus focused to reduce interaction cost.

---

## 2. Feedback Latency Thresholds (Miller & Nielsen)

Human cognitive perception operates on three distinct temporal thresholds:
- **0.1 second (100 ms)**: Limit for feeling that the system is reacting instantaneously. No special feedback needed other than the direct visual state change (e.g. button press depression).
- **1.0 second (1000 ms)**: Limit for the user's flow of thought to remain uninterrupted. The user notices the delay, but feels in control. Requires a subtle inline activity spinner.
- **10 seconds**: Limit for keeping the user's attention focused on the dialogue. Beyond 10 seconds, users switch tasks. Requires a percentage-determinate progress bar with estimated remaining time and an explicit cancel option.

---

## 3. Micro-Interaction Choreography (Dan Saffer Framework)

```
[Trigger] ──► [Rules] ──► [Feedback] ──► [Loops & Modes]
```

1. **Trigger**: Initiates the micro-interaction. Can be user-initiated (click, hover, swipe) or system-initiated (threshold exceeded, message arrived).
2. **Rules**: Determine what can and cannot happen. They define the state machine behind the interaction.
3. **Feedback**: Informs the user what is happening through visual, audio, or haptic signals. Must be immediate and unambiguous.
4. **Loops & Modes**: Determine interaction duration and state persistence (e.g., toggle state, countdown timer, auto-dismiss loop).

---

## 4. Animation Easing & Spring Physics

### 4.1 Cubic Bezier Acceleration Curves
CSS animations are defined by a 2D cubic Bezier curve $P(t) = (1-t)^3 P_0 + 3(1-t)^2 t P_1 + 3(1-t) t^2 P_2 + t^3 P_3$:
- **Standard (Natural deceleration at end)**: `cubic-bezier(0.2, 0.0, 0.0, 1.0)`
- **Entrance (Fast in, gentle deceleration)**: `cubic-bezier(0.0, 0.0, 0.2, 1.0)`
- **Exit (Slow start, accelerates off screen)**: `cubic-bezier(0.4, 0.0, 1.0, 1.0)`

### 4.2 Spring Physics
Unlike fixed-duration Bezier curves, springs model physical forces (mass, stiffness, damping):
$$F = -kx - c v$$
- $k$: Stiffness (spring rate).
- $c$: Damping coefficient (friction).
- Advantage: Naturally handles interruptions. If a user swipes a drawer mid-animation, a spring preserves existing momentum rather than restarting an arbitrary timer.
