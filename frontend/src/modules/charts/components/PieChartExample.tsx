const data = [
  { name: '分类A', value: 400 },
  { name: '分类B', value: 300 },
  { name: '分类C', value: 300 },
  { name: '分类D', value: 200 },
  { name: '分类E', value: 100 },
];

const COLORS = ['#8884d8', '#83a6ed', '#8dd1e1', '#82ca9d', '#a4de6c'];

export function PieChartExample() {
  const total = data.reduce((sum, item) => sum + item.value, 0);
  const radius = 78;
  const center = 150;
  let offset = 0;

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-3">
      <svg viewBox="0 0 300 230" className="h-[78%] w-full">
        {data.map((item, index) => {
          const value = item.value / total;
          const start = offset * Math.PI * 2 - Math.PI / 2;
          offset += value;
          const end = offset * Math.PI * 2 - Math.PI / 2;
          const largeArc = value > 0.5 ? 1 : 0;
          const x1 = center + radius * Math.cos(start);
          const y1 = 108 + radius * Math.sin(start);
          const x2 = center + radius * Math.cos(end);
          const y2 = 108 + radius * Math.sin(end);
          const labelAngle = (start + end) / 2;
          const labelX = center + (radius + 34) * Math.cos(labelAngle);
          const labelY = 108 + (radius + 24) * Math.sin(labelAngle);

          return (
            <g key={item.name}>
              <path d={`M ${center} 108 L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2} Z`} fill={COLORS[index]} />
              <text x={labelX} y={labelY} textAnchor="middle" className="fill-[var(--color-text-secondary)] text-[11px]">
                {Math.round(value * 100)}%
              </text>
            </g>
          );
        })}
        <circle cx={center} cy="108" r="42" fill="var(--color-bg-surface)" />
        <text x={center} y="104" textAnchor="middle" className="fill-[var(--color-text-primary)] text-[18px] font-semibold">
          {total}
        </text>
        <text x={center} y="124" textAnchor="middle" className="fill-[var(--color-text-muted)] text-[11px]">
          总量
        </text>
      </svg>
      <div className="flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-[var(--color-text-secondary)]">
        {data.map((item, index) => (
          <span key={item.name} className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: COLORS[index] }} />
            {item.name}
          </span>
        ))}
      </div>
    </div>
  );
}
