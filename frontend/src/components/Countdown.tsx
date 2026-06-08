import { useState, useEffect, useRef } from 'react';

interface Props {
  endTimestampMs: number;
  serverTimeMs: number;
  onEnd?: () => void;
}

function DigitBox({ ch, urgent }: { ch: string; urgent?: boolean }) {
  return (
    <span style={{
      display: 'inline-block', width: 26, height: 32, lineHeight: '32px',
      background: urgent ? '#7f1d1d' : '#1a202c',
      color: urgent ? '#fca5a5' : '#fff', borderRadius: 6,
      textAlign: 'center', fontSize: 18, fontFamily: 'monospace', fontWeight: 'bold',
      margin: '0 1px',
      transition: 'background 0.3s, color 0.3s',
    }}>{ch}</span>
  );
}

export default function Countdown({ endTimestampMs, serverTimeMs, onEnd }: Props) {
  const [remaining, setRemaining] = useState(0);
  const onEndRef = useRef(onEnd);
  onEndRef.current = onEnd;

  useEffect(() => {
    if (!endTimestampMs) return;

    // Use absolute timestamp — no offset resets
    const tick = () => {
      const ms = endTimestampMs - Date.now();
      if (ms <= 0) {
        setRemaining(0);
        onEndRef.current?.();
        return;
      }
      setRemaining(ms);
    };

    tick();
    const timer = setInterval(tick, 200);
    return () => clearInterval(timer);
  }, [endTimestampMs]);

  const totalSec = Math.floor(remaining / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  const digits = String(min).padStart(2, '0') + String(sec).padStart(2, '0');
  const urgent = totalSec <= 10 && remaining > 0;

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 0 }}>
      <DigitBox ch={digits[0]} urgent={urgent} />
      <DigitBox ch={digits[1]} urgent={urgent} />
      <span style={{ color: urgent ? '#fca5a5' : '#fff', fontWeight: 'bold', fontSize: 16, margin: '0 2px', transition: 'color 0.3s' }}>:</span>
      <DigitBox ch={digits[2]} urgent={urgent} />
      <DigitBox ch={digits[3]} urgent={urgent} />
    </span>
  );
}
