import { Stack, Typography, Box, Button, Divider, Select, MenuItem, FormControl } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import SubCard from 'ui-component/cards/SubCard';
import { IconMessageChatbot } from '@tabler/icons-react';
import { useSelector } from 'react-redux';
import { useState, useCallback } from 'react';
import { API } from 'utils/api';
import { replaceChatPlaceholders, getChatLinks } from 'utils/common';
import useRelayEndpoint from 'hooks/useRelayEndpoint';
import EndpointStatus from 'ui-component/EndpointStatus';
import { IconAppWindow } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

const QuickStartCard = () => {
  const { t } = useTranslation();
  const [key, setKey] = useState('');
  const theme = useTheme();
  const siteInfo = useSelector((state) => state.siteInfo);
  const chatLinks = getChatLinks(false);
  const { endpoints: relayEndpoints, endpoint: relayEndpoint, url: relayUrl, choose: chooseRelayEndpoint } = useRelayEndpoint();
  const baseServer = relayUrl || siteInfo.server_address;

  const getProcessedUrl = useCallback(
    (url, key) => {
      let server = baseServer || window.location.host;
      server = encodeURIComponent(server);
      const useKey = 'sk-' + key;
      return replaceChatPlaceholders(url, useKey, server);
    },
    [baseServer]
  );

  const handleClick = async (url) => {
    if (!key) {
      try {
        const res = await API.get(`/api/token/playground`);
        const { success, message, data } = res.data;
        if (success) {
          setKey(data);
          window.open(getProcessedUrl(url, data), '_blank');
        } else {
          console.log('message', message);
        }
      } catch (error) {
        console.error('Failed to get token:', error);
      }
    } else {
      window.open(getProcessedUrl(url, key), '_blank');
    }
  };

  return (
    <Box>
      <SubCard>
        <Stack spacing={3}>
          <Typography variant="h3" color={theme.palette.primary.dark}>
            {t('dashboard_index.quickStart')}
          </Typography>
          <Typography variant="body1" color="textSecondary">
            {t('dashboard_index.quickStartTip')}
          </Typography>

          {relayEndpoints.length > 1 && (
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="body2" color="textSecondary">
                当前线路
              </Typography>
              <FormControl size="small" sx={{ minWidth: 140 }}>
                <Select
                  value={relayEndpoint?.id || ''}
                  onChange={(event) => {
                    const next = relayEndpoints.find((item) => item.id === event.target.value);
                    if (next) chooseRelayEndpoint(next);
                  }}
                >
                  {relayEndpoints.map((item) => (
                    <MenuItem key={item.id} value={item.id}>
                      {item.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <Typography variant="caption" color="textSecondary">
                {relayUrl}
              </Typography>
              <EndpointStatus endpoint={relayEndpoint} />
            </Stack>
          )}

          <Stack direction="row" flexWrap="wrap" sx={{ gap: 2 }}>
            {chatLinks.map(
              (option, index) =>
                option.url.startsWith('http') && (
                  <Button
                    key={index}
                    variant="contained"
                    startIcon={<IconMessageChatbot />}
                    onClick={() => handleClick(option.url)}
                    sx={{
                      backgroundColor: theme.palette.primary.main,
                      color: 'white',
                      '&:hover': {
                        backgroundColor: theme.palette.primary.dark,
                        boxShadow: '0 0 3px 0 rgba(11, 108, 235, 0.5)'
                      },
                      textTransform: 'none',
                      boxShadow: '0 0 3px 0 rgba(11, 108, 235, 0.5)'
                    }}
                  >
                    {option.name}
                  </Button>
                )
            )}
          </Stack>
          <Divider />
          <Stack direction="row" flexWrap="wrap" sx={{ gap: 2 }}>
            {chatLinks.map(
              (option, index) =>
                !option.url.startsWith('http') && (
                  <Button
                    key={index}
                    variant="contained"
                    startIcon={<IconAppWindow />}
                    onClick={() => handleClick(option.url)}
                    sx={{
                      backgroundColor: '#00B8D4',
                      color: 'white',
                      '&:hover': {
                        backgroundColor: '#00838F',
                        boxShadow: '0 0 3px 0 #00838F'
                      },
                      textTransform: 'none',
                      boxShadow: '0 0 3px 0 #00838F'
                    }}
                  >
                    {option.name}
                  </Button>
                )
            )}
          </Stack>
        </Stack>
      </SubCard>
    </Box>
  );
};

export default QuickStartCard;
