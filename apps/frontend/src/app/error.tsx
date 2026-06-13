'use client';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div style={{ padding: '2rem', fontFamily: 'monospace', background: '#111', color: '#0f0', minHeight: '100vh' }}>
      <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem' }}>Something went wrong!</h2>
      <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', background: '#000', padding: '1rem', borderRadius: '8px', border: '1px solid #333' }}>
        {error.message}
        {'\n\n'}
        {error.stack}
      </pre>
      <button
        onClick={reset}
        style={{ marginTop: '1rem', padding: '0.5rem 1rem', background: '#0f0', color: '#000', border: 'none', cursor: 'pointer', fontWeight: 'bold' }}
      >
        Try again
      </button>
    </div>
  );
}
