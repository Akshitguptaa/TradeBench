import React, { useState } from 'react';
import { UploadCloud, CheckCircle2, XCircle, Loader2, Play, FileText } from 'lucide-react';
import { useSubmitCode } from '../hooks/useSubmitCode';

export function SubmitForm() {
  const { status, error, submitCode, score, reset } = useSubmitCode();
  const [contestantId, setContestantId] = useState('');
  const [language, setLanguage] = useState('python');
  const [file, setFile] = useState<File | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!contestantId || !file) return;
    submitCode(contestantId, language, file);
  };

  const isWorking = ['submitting', 'queued', 'running'].includes(status);

  return (
    <div className="w-full bg-white neo-border neo-shadow">
      {/* Header */}
      <div className="px-6 py-4 border-b-3 border-black bg-[var(--color-neo-blue)]">
        <h2 className="text-xl font-black uppercase tracking-tight">
          Submit Code
        </h2>
      </div>
      
      {/* Body */}
      <div className="p-6">
        {status === 'completed' ? (
          <div className="flex flex-col items-center justify-center p-8 bg-[var(--color-neo-green)] neo-border space-y-4 text-center">
            <CheckCircle2 size={56} strokeWidth={2.5} />
            <h3 className="text-2xl font-black uppercase">Success!</h3>
            <div className="text-xl font-bold bg-white px-5 py-2 neo-border">
              Score: {score?.toFixed(2) || 'N/A'}
            </div>
            <button 
              onClick={reset}
              className="mt-4 w-full py-3 bg-white font-black uppercase tracking-widest text-sm neo-button"
            >
              Submit Another
            </button>
          </div>
        ) : status === 'error' || status === 'failed' ? (
          <div className="flex flex-col items-center justify-center p-8 bg-[var(--color-neo-pink)] neo-border space-y-4 text-center">
            <XCircle size={56} strokeWidth={2.5} />
            <h3 className="text-2xl font-black uppercase">Failed</h3>
            <div className="text-sm font-bold bg-white px-5 py-3 neo-border break-words w-full">
              {error || 'Unknown error occurred'}
            </div>
            <button 
              onClick={reset}
              className="mt-4 w-full py-3 bg-white font-black uppercase tracking-widest text-sm neo-button"
            >
              Try Again
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* Contestant ID */}
            <div className="space-y-2">
              <label className="block text-xs font-black uppercase tracking-wider text-black/60">
                Contestant ID
              </label>
              <input 
                type="text" 
                required
                disabled={isWorking}
                value={contestantId}
                onChange={(e) => setContestantId(e.target.value)}
                className="w-full px-4 py-3 text-base font-bold bg-white neo-border focus:outline-none focus:shadow-[3px_3px_0px_0px_rgba(0,0,0,1)] focus:-translate-y-0.5 transition-all placeholder:text-black/25"
                placeholder="e.g. alice_123"
              />
            </div>

            {/* Language */}
            <div className="space-y-2">
              <label className="block text-xs font-black uppercase tracking-wider text-black/60">
                Language
              </label>
              <select 
                value={language}
                disabled={isWorking}
                onChange={(e) => setLanguage(e.target.value)}
                className="w-full px-4 py-3 text-base font-bold bg-white neo-border neo-select focus:outline-none focus:shadow-[3px_3px_0px_0px_rgba(0,0,0,1)] focus:-translate-y-0.5 transition-all cursor-pointer appearance-none"
              >
                <option value="python">Python</option>
                <option value="cpp">C++</option>
                <option value="go">Go</option>
                <option value="rust">Rust</option>
              </select>
            </div>

            {/* File Upload */}
            <div className="space-y-2">
              <label className="block text-xs font-black uppercase tracking-wider text-black/60">
                Source Code
              </label>
              <label className={`flex flex-col items-center justify-center w-full py-8 neo-border cursor-pointer transition-all hover:-translate-y-0.5 hover:shadow-[3px_3px_0px_0px_rgba(0,0,0,1)] ${
                file 
                  ? 'bg-[var(--color-neo-green)]' 
                  : 'bg-[var(--background)] hover:bg-[var(--color-neo-blue)]/30'
              }`}>
                <div className="flex flex-col items-center gap-2 px-4 text-center">
                  {file ? (
                    <>
                      <FileText size={32} strokeWidth={2} />
                      <p className="text-sm font-bold truncate max-w-full">{file.name}</p>
                    </>
                  ) : (
                    <>
                      <UploadCloud size={32} strokeWidth={2} className="text-black/40" />
                      <p className="text-sm font-bold text-black/40">Click to upload</p>
                    </>
                  )}
                </div>
                <input 
                  type="file" 
                  className="hidden" 
                  required
                  disabled={isWorking}
                  onChange={(e) => setFile(e.target.files?.[0] || null)}
                />
              </label>
            </div>

            {/* Submit Button */}
            <button 
              type="submit" 
              disabled={isWorking || !file || !contestantId}
              className="w-full py-3.5 mt-2 text-base font-black uppercase tracking-widest bg-[var(--color-neo-yellow)] neo-button disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {isWorking ? (
                <>
                  <Loader2 className="animate-spin" size={20} strokeWidth={3} />
                  {status.toUpperCase()}...
                </>
              ) : (
                <>
                  <Play size={18} className="fill-black" />
                  Submit Run
                </>
              )}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
