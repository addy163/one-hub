import PropTypes from 'prop-types';
import { Box, Stack, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { getEndpointHealth, getEndpointStatusText } from 'utils/endpoints';

// 线路健康状态小圆点 + 文案，用于令牌页、仪表盘与后台线路表格。
export default function EndpointStatus({ endpoint, showText = true, size = 8 }) {
  const theme = useTheme();
  const { status } = getEndpointHealth(endpoint);

  const color =
    status === 'up' ? theme.palette.success.main : status === 'down' ? theme.palette.error.main : theme.palette.text.disabled;

  return (
    <Stack direction="row" alignItems="center" spacing={0.75}>
      <Box
        sx={{
          width: size,
          height: size,
          borderRadius: '50%',
          backgroundColor: color,
          boxShadow: status === 'up' ? `0 0 0 3px ${theme.palette.success.light}22` : 'none',
          flexShrink: 0
        }}
      />
      {showText && (
        <Typography variant="caption" color="text.secondary">
          {getEndpointStatusText(endpoint)}
        </Typography>
      )}
    </Stack>
  );
}

EndpointStatus.propTypes = {
  endpoint: PropTypes.object,
  showText: PropTypes.bool,
  size: PropTypes.number
};
