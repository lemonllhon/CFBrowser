import type { ReactNode } from 'react';

type LegendItem = {
  label: string;
  color: string;
}

export type ChartPoint = {
  name: string;
  [key: string]: number | string;
}

const WIDTH = 720;
const HEIGHT = 300;
const PADDING = { top: 24, right: 28, bottom: 44, left: 54 };

export const plot = {
  width: WIDTH,
  height: HEIGHT,
  left: PADDING.left,
  top: PADDING.top,
  right: WIDTH - PADDING.right,
  bottom: HEIGHT - PADDING.bottom,
  innerWidth: WIDTH - PADDING.left - PADDING.right,
  innerHeight: HEIGHT - PADDING.top - PADDING.bottom,
}

export function valueOf(point: ChartPoint, key: string) {
  return Number(point[key] || 0);
}

export function maxValue(data: ChartPoint[], keys: string[]) {
  return Math.max(1, ...data.flatMap((point) => keys.map((key) => valueOf(point, key))));
}

export function xAt(index: number, length: number) {
  if (length <= 1) return plot.left + plot.innerWidth / 2;
  return plot.left + (index / (length - 1)) * plot.innerWidth;
}

export function yAt(value: number, max: number) {
  return plot.bottom - (value / max) * plot.innerHeight;
}

export function pointsFor(data: ChartPoint[], key: string, max: number) {
  return data.map((point, index) => ({
    x: xAt(index, data.length),
    y: yAt(valueOf(point, key), max),
  }));
}

export function pathFor(points: Array<{ x: number; y: number }>) {
  return points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x.toFixed(1)} ${point.y.toFixed(1)}`).join(' ');
}

export function areaPathFor(points: Array<{ x: number; y: number }>) {
  if (!points.length) return '';
  const last = points[points.length - 1];
  return `${pathFor(points)} L ${last.x.toFixed(1)} ${plot.bottom} L ${points[0].x.toFixed(1)} ${plot.bottom} Z`;
}

export function SvgChart({
  data,
  max,
  legend,
  children,
}: {
  data: ChartPoint[];
  max: number;
  legend: LegendItem[];
  children: ReactNode;
}) {
  const ticks = [0, 0.25, 0.5, 0.75, 1];

  return (
    <div className="relative h-full w-full">
      <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} className="h-full w-full overflow-visible">
        {ticks.map((tick) => {
          const y = plot.bottom - tick * plot.innerHeight;
          return (
            <g key={tick}>
              <line x1={plot.left} x2={plot.right} y1={y} y2={y} stroke="var(--color-border-muted)" strokeDasharray="4 4" />
              <text x={plot.left - 10} y={y + 4} textAnchor="end" className="fill-[var(--color-text-muted)] text-[11px]">
                {Math.round(max * tick)}
              </text>
            </g>
          );
        })}
        <line x1={plot.left} x2={plot.right} y1={plot.bottom} y2={plot.bottom} stroke="var(--color-border-default)" />
        <line x1={plot.left} x2={plot.left} y1={plot.top} y2={plot.bottom} stroke="var(--color-border-default)" />
        {data.map((point, index) => (
          <text key={point.name} x={xAt(index, data.length)} y={plot.bottom + 24} textAnchor="middle" className="fill-[var(--color-text-secondary)] text-[12px]">
            {point.name}
          </text>
        ))}
        {children}
      </svg>
      <div className="absolute bottom-0 left-0 right-0 flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-[var(--color-text-secondary)]">
        {legend.map((item) => (
          <span key={item.label} className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: item.color }} />
            {item.label}
          </span>
        ))}
      </div>
    </div>
  );
}
