/**
 * The CPTC mark with the optional "CPTC <product>" wordmark for navbars.
 *
 * The image is passed in rather than imported here because Next and Vite resolve static image
 * imports differently (Next gives an object, Vite a URL). Copy `@cptc/ui/assets/cptc-mark.webp`
 * into the app's public folder and pass `/cptc-mark.webp`, or pass your bundler's URL.
 */
export function BrandMark({
  src,
  size = 32,
  product,
  className,
}: {
  src: string;
  size?: number;
  /** Shown after "CPTC" in color-fg-muted, e.g. "Volunteer Management Portal". Omit for the mark alone. */
  product?: string;
  className?: string;
}) {
  return (
    <span className={className ?? 'inline-flex items-center gap-2.5'}>
      <img src={src} alt="" width={size} height={size} className="shrink-0" />
      {product ? (
        <span className="hidden items-baseline gap-1.5 sm:flex">
          <span className="text-base font-extrabold tracking-tight text-fg">CPTC</span>
          <span className="text-sm font-medium text-fg-muted">{product}</span>
        </span>
      ) : null}
    </span>
  );
}
