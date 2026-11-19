export function SolaceMark({ size = 22 }: { size?: number }) {
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path d="M12 2.2c.2 1.3.7 1.8 2 2-.3.1-.6.2-.8.3-.7.3-1 .8-1.2 1.7-.2-1.3-.7-1.8-2-2 1.3-.2 1.8-.7 2-2Z" fill="currentColor"/>
    <path d="M3.5 7.6c3.1-1.1 6-.5 8 1.4v10.6c-2.1-1.6-4.8-2.1-8-1.1V7.6Zm17 0c-3.1-1.1-6-.5-8 1.4v10.6c2.1-1.6 4.8-2.1 8-1.1V7.6Z" stroke="currentColor" strokeWidth="1.55" strokeLinejoin="round"/>
    <path d="M12 9.3v10" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" opacity=".55"/>
  </svg>
}
