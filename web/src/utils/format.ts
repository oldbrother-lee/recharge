/**
 * 格式化运营商字段
 * @param isp 运营商代码，格式如 "1,2,3"
 * @returns 格式化后的运营商名称，如 "移动、电信、联通"
 */
export const formatISP = (isp: string): string => {
  if (!isp) return '';
  
  const ispMap: Record<string, string> = {
    '1': '移动',
    '2': '电信',
    '3': '联通'
  };
  
  return isp.split(',')
    .map(code => ispMap[code] || code)
    .join('、');
}; 