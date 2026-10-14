import * as echarts from 'echarts/core';
import { LineChart, PieChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, DataZoomComponent } from 'echarts/components';
import { SVGRenderer } from 'echarts/renderers';
echarts.use([LineChart, PieChart, GridComponent, TooltipComponent, DataZoomComponent, SVGRenderer]);
export { echarts };
export const colors: Record<string, string> = { commodities: '#bf8b36', bonds: '#5470c6', money_market_funds: '#73a0ac', stocks: '#219773', real_estate: '#9b6fc3', cryptocurrencies: '#ee8a43', altcoins: '#d65f85', cash: '#8895a7', unclassified: '#454e59' };
// This conversion is exclusively for chart geometry. Labels always use Go strings.
export function coordinate(value: string): number | null {
  const n = Number(value);
  return Number.isFinite(n) && (n !== 0 || !/[1-9]/.test(value)) ? n : null;
}
