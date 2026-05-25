import { areaPathFor, maxValue, pathFor, pointsFor, SvgChart } from './SvgChart';

const data = [
  { name: '周一', 系统A: 4000, 系统B: 2400, 系统C: 1800 },
  { name: '周二', 系统A: 3000, 系统B: 1398, 系统C: 2300 },
  { name: '周三', 系统A: 2000, 系统B: 9800, 系统C: 2500 },
  { name: '周四', 系统A: 2780, 系统B: 3908, 系统C: 1908 },
  { name: '周五', 系统A: 1890, 系统B: 4800, 系统C: 2800 },
  { name: '周六', 系统A: 2390, 系统B: 3800, 系统C: 3200 },
  { name: '周日', 系统A: 3490, 系统B: 4300, 系统C: 2100 },
];

export function AreaChartExample() {
  const series = [
    { key: '系统A', color: '#8884d8' },
    { key: '系统B', color: '#82ca9d' },
    { key: '系统C', color: '#ffc658' },
  ];
  const max = maxValue(data, series.map((item) => item.key));

  return (
    <SvgChart data={data} max={max} legend={series.map(({ key, color }) => ({ label: key, color }))}>
      {series.map(({ key, color }) => {
        const points = pointsFor(data, key, max);
        return (
          <g key={key}>
            <path d={areaPathFor(points)} fill={color} opacity="0.2" />
            <path d={pathFor(points)} fill="none" stroke={color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
          </g>
        );
      })}
    </SvgChart>
  );
}
