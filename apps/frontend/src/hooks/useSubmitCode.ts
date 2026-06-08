import { useState, useCallback, useRef, useEffect } from 'react';

export type SubmissionStatus = 'idle' | 'submitting' | 'queued' | 'running' | 'completed' | 'failed' | 'error';

export function useSubmitCode() {
  const [status, setStatus] = useState<SubmissionStatus>('idle');
  const [error, setError] = useState<string | null>(null);
  const [submissionId, setSubmissionId] = useState<string | null>(null);
  const [score, setScore] = useState<number | null>(null);
  
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const clearPolling = () => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  };

  const pollStatus = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/v1/submissions/${id}`);
      if (!res.ok) {
        if (res.status === 404) return; // Might not be available yet
        throw new Error('Failed to fetch status');
      }
      const data = await res.json();
      
      setStatus(data.status);
      
      if (data.status === 'completed' || data.status === 'failed') {
        clearPolling();
        if (data.score !== undefined) {
          setScore(data.score);
        }
      }
    } catch (err) {
      console.error('Polling error:', err);
    }
  }, []);

  const submitCode = useCallback(async (contestantId: string, language: string, file: File) => {
    setStatus('submitting');
    setError(null);
    setScore(null);
    setSubmissionId(null);
    clearPolling();

    const formData = new FormData();
    formData.append('contestant_id', contestantId);
    formData.append('language', language);
    formData.append('code', file);

    try {
      const res = await fetch('/api/v1/submissions', {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || 'Submission failed');
      }

      const data = await res.json();
      const id = data.submission_id || data.id;
      setSubmissionId(id);
      setStatus('queued');

      // Start polling
      pollIntervalRef.current = setInterval(() => {
        pollStatus(id);
      }, 2000);
      
    } catch (err: any) {
      setStatus('error');
      setError(err.message || 'An error occurred');
    }
  }, [pollStatus]);

  useEffect(() => {
    return () => clearPolling();
  }, []);

  return {
    status,
    error,
    submissionId,
    score,
    submitCode,
    reset: () => {
      setStatus('idle');
      setError(null);
      setSubmissionId(null);
      setScore(null);
      clearPolling();
    }
  };
}
