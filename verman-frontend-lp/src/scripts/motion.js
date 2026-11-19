// Progressive enhancement: content stays visible when JavaScript is unavailable.
const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
const running = new Set();
const byElement = new WeakMap();
const easing = "cubic-bezier(0.22, 1, 0.36, 1)";

export function animateContent(
  element,
  { delay = 0, distance = 20, duration = 650 } = {},
) {
  if (!element || preference.matches || !element.animate) return;
  byElement.get(element)?.cancel();
  const animation = element.animate(
    [
      { opacity: 0, translate: `0 ${distance}px` },
      { opacity: 1, translate: "0 0" },
    ],
    { duration, delay, easing, fill: "backwards" },
  );
  byElement.set(element, animation);
  running.add(animation);
  const release = () => {
    running.delete(animation);
    if (byElement.get(element) === animation) byElement.delete(element);
  };
  animation.addEventListener("finish", release, { once: true });
  animation.addEventListener("cancel", release, { once: true });
}

export function initMotion() {
  if (!("IntersectionObserver" in window)) return;
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach(({ target, isIntersecting }) => {
        if (!isIntersecting) return;
        observer.unobserve(target);
        // Keyboard navigation and anchor targets should be readable immediately.
        if (
          target.contains(document.activeElement) ||
          target.matches(":target")
        )
          return;
        animateContent(target, {
          delay: Number(target.dataset.motionDelay) || 0,
        });
      });
    },
    { threshold: 0.08, rootMargin: "0px 0px -24px 0px" },
  );

  const groups = [
    [".hero-copy", ":scope > *"],
    [".hero-grid", ":scope > .hero-visual"],
    [".hero", ":scope > .hero-bottom"],
    ["main", ".section-heading"],
    [".tools-list", ":scope > .tool-chip"],
    [".tool-showcase", ":scope > .terminal-panel"],
    [".features-grid", ":scope > .feature"],
    [
      ".desktop-section > .container",
      ":scope > .gallery-tabs, :scope > .desktop-frame, :scope > .gallery-caption, :scope > .capability-list",
    ],
    [".steps-grid", ":scope > .step"],
    [".path-grid, .docs-promo, .faq-grid, .download-inner", ":scope > *"],
    [".site-footer", ".footer-top, .footer-bottom"],
  ];
  const registered = new Set();
  groups.forEach(([parentSelector, childSelector]) => {
    document.querySelectorAll(parentSelector).forEach((parent) => {
      parent.querySelectorAll(childSelector).forEach((element, index) => {
        if (registered.has(element)) return;
        registered.add(element);
        element.dataset.motionDelay = String(Math.min(index, 5) * 55);
        observer.observe(element);
      });
    });
  });

  document.addEventListener("focusin", (event) => {
    let element = event.target;
    while (element instanceof Element) {
      byElement.get(element)?.cancel();
      element = element.parentElement;
    }
  });
  preference.addEventListener("change", () => {
    if (preference.matches) running.forEach((animation) => animation.cancel());
  });
  window.addEventListener(
    "pagehide",
    () => {
      observer.disconnect();
      running.forEach((animation) => animation.cancel());
    },
    { once: true },
  );
}
