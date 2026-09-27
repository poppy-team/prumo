# Marketing Collateral Production Guide

## 1. Funnel Alignment
Effective marketing collateral must map to the customer journey:
- **Awareness (Top of Funnel):** Goal is reach and education. Formats: Social graphics, infographics, short-form video, blog post images.
- **Consideration (Middle of Funnel):** Goal is evaluation and trust-building. Formats: Case studies, whitepapers, comparison charts, webinar slides.
- **Conversion (Bottom of Funnel):** Goal is action and sales. Formats: Pitch decks, sell sheets, pricing guides, highly optimized landing pages.
- **Retention (Post-Purchase):** Goal is loyalty and advocacy. Formats: Onboarding emails, newsletters, exclusive community assets.

## 2. Technical Specifications by Channel

### Social Media
Maintaining exact dimensions prevents cropping and pixelation.
- **Instagram:**
  - Square: 1080 x 1080 px (1:1)
  - Portrait: 1080 x 1350 px (4:5)
  - Story/Reels: 1080 x 1920 px (9:16)
- **Facebook:**
  - Link Post: 1200 x 630 px (1.9:1)
- **LinkedIn:**
  - Single Image: 1200 x 627 px (1.91:1)
- **Twitter/X:**
  - In-Stream Photo: 1200 x 675 px (16:9)

### Email Marketing
Email clients have notorious rendering quirks.
- **Width:** Maximum 600px to ensure mobile and desktop compatibility.
- **Styling:** Use inline CSS. Many clients strip out `<style>` blocks in the `<head>`.
- **Images:** Always use `alt` text. Background images may not render in Outlook.
- **Accessibility:** High contrast ratios, semantic HTML.

### Landing Pages
Landing pages should be self-contained conversion engines.
- **Structure:**
  1. Hero section (Value prop + CTA).
  2. Social proof (Logos, testimonials).
  3. Benefits (Not just features).
  4. FAQ (Objection handling).
  5. Final CTA block.

### Print Media
Physical production requires strict adherence to technical standards.
- **Color:** CMYK mode (Cyan, Magenta, Yellow, Key/Black).
- **Resolution:** 300 DPI (Dots Per Inch) minimum for all raster images.
- **Bleed:** 3mm (or 0.125 inches) extending past the trim edge to prevent white margins after cutting.
- **Export:** PDF/X-1a is the safest standard for commercial printing.

## 3. Digital Asset Management (DAM)
- Use standard naming conventions: `YYYYMMDD_CampaignName_AssetType_Size_Version.ext`
- Example: `20260927_FallPromo_IGStory_1080x1920_v2.png`
