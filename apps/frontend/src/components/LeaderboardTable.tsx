import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useLeaderboard } from '../hooks/useLeaderboard';
import { Trophy, Zap } from 'lucide-react';

export function LeaderboardTable() {
  const { entries, isConnected } = useLeaderboard();

  const maxScore = entries.length > 0 ? Math.max(...entries.map(e => e.score)) : 100;
  const safeMax = maxScore > 0 ? maxScore : 1;

  const getRankDisplay = (rank: number) => {
    switch (rank) {
      case 1: return <span className="text-xl">🥇</span>;
      case 2: return <span className="text-xl">🥈</span>;
      case 3: return <span className="text-xl">🥉</span>;
      default: return <span className="text-sm font-black text-black/30">#{rank}</span>;
    }
  };

  return (
    <div className="w-full bg-white neo-border neo-shadow flex flex-col">
      {/* Header */}
      <div className="px-6 py-4 border-b-3 border-black bg-[var(--color-neo-pink)] flex items-center justify-between">
        <h2 className="text-xl font-black uppercase tracking-tight flex items-center gap-2">
          <Trophy size={22} strokeWidth={2.5} />
          Leaderboard
        </h2>
        <div className="flex items-center gap-2 bg-white neo-border px-3 py-1.5">
          <span className={`w-2.5 h-2.5 rounded-full border-2 border-black ${isConnected ? 'bg-green-400 animate-pulse' : 'bg-red-500'}`} />
          <span className="text-xs font-black uppercase tracking-wider">
            {isConnected ? 'Live' : 'Off'}
          </span>
        </div>
      </div>
      
      {/* Table */}
      <div className="flex flex-col">
        {/* Table Header */}
        <div className="hidden sm:grid grid-cols-[56px_1fr_160px_100px] items-center px-6 py-3 border-b-3 border-black bg-[var(--background)] text-xs font-black uppercase tracking-wider text-black/50">
          <div className="text-center">#</div>
          <div>Contestant</div>
          <div className="text-right">Score</div>
          <div className="text-right">p99</div>
        </div>

        {/* Table Body */}
        <div className="neo-scroll overflow-y-auto max-h-[calc(100vh-240px)]">
          <AnimatePresence>
            {entries.map((entry, index) => (
              <motion.div
                key={entry.contestant_id}
                layout
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, transition: { duration: 0.15 } }}
                transition={{ type: "spring", stiffness: 500, damping: 30 }}
                className={`grid grid-cols-[56px_1fr] sm:grid-cols-[56px_1fr_160px_100px] items-center px-6 py-3.5 border-b-2 border-black/10 hover:bg-[var(--color-neo-yellow)]/20 transition-colors ${
                  index < 3 ? 'bg-[var(--color-neo-yellow)]/10' : ''
                }`}
              >
                {/* Rank */}
                <div className="flex items-center justify-center">
                  {getRankDisplay(entry.rank)}
                </div>
                
                {/* Contestant Name */}
                <div className="font-bold text-base truncate pr-4">
                  {entry.contestant_id}
                </div>
                
                {/* Score + Bar */}
                <div className="col-span-2 sm:col-span-1 flex items-center justify-end gap-3 mt-1 sm:mt-0">
                  <span className="font-black text-base tabular-nums">{entry.score.toFixed(2)}</span>
                  <div className="w-16 h-3.5 bg-[var(--background)] neo-border overflow-hidden">
                    <motion.div 
                      className="h-full bg-[var(--color-neo-yellow)]"
                      initial={{ width: 0 }}
                      animate={{ width: `${(entry.score / safeMax) * 100}%` }}
                      transition={{ type: "spring", stiffness: 80, damping: 15 }}
                    />
                  </div>
                </div>
                
                {/* Latency */}
                <div className="hidden sm:flex items-center justify-end">
                  {entry.p99_ms ? (
                    <span className="font-mono text-sm font-bold flex items-center gap-1">
                      <Zap size={14} className={entry.p99_ms < 50 ? 'fill-[var(--color-neo-yellow)] text-black' : 'text-black/30'} />
                      {entry.p99_ms.toFixed(1)}ms
                    </span>
                  ) : (
                    <span className="text-sm text-black/20 font-bold">—</span>
                  )}
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
          
          {entries.length === 0 && (
            <div className="flex flex-col items-center justify-center py-24 text-black/30 space-y-3">
              <Trophy size={56} strokeWidth={1.5} />
              <p className="font-black uppercase tracking-widest text-base text-center">
                No entries yet
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
