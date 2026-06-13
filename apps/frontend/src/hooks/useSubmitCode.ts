import { useState, useCallback, useRef, useEffect } from 'react';

export type SubmissionStatus = 'idle' | 'submitting' | 'queued' | 'running' | 'completed' | 'failed' | 'error';

export function useSubmitCode() {
  const [status, setStatus] = useState<SubmissionStatus>('idle');
  const [error, setError] = useState<string | null>(null);
  const [submissionId, setSubmissionId] = useState<string | null>(null);
  const [score, setScore] = useState<number | null>(null);

  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const pollTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const clearPolling = () => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
    if (pollTimeoutRef.current) {
      clearTimeout(pollTimeoutRef.current);
      pollTimeoutRef.current = null;
    }
  };

  const pollStatus = useCallback(async (id: string, token: string) => {
    try {
      const res = await fetch(`/api/v1/submissions/${id}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
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
    formData.append('file', file);

    try {
      // 1. Authenticate to get a token for the gateway
      const tokenRes = await fetch('/api/v1/auth/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contestant_id: contestantId })
      });

      if (!tokenRes.ok) {
        throw new Error('Failed to authenticate');
      }

      const tokenData = await tokenRes.json();
      const token = tokenData.token;

      // 2. Submit the code with the token
      const res = await fetch('/api/v1/submissions', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
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

      // Start polling (1s interval for snappy updates)
      pollIntervalRef.current = setInterval(() => {
        pollStatus(id, token);
      }, 1000);

      // Safety timeout: if submission doesn't complete within 3 minutes,
      // stop polling and report failure so the UI doesn't spin forever.
      pollTimeoutRef.current = setTimeout(() => {
        clearPolling();
        setStatus('failed');
        setError('Submission timed out — the run did not complete within 3 minutes. Please try again.');
      }, 180_000);

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
