import PropTypes from 'prop-types';
import * as Yup from 'yup';
import { Formik } from 'formik';
import { useTheme } from '@mui/material/styles';
import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormHelperText,
  InputAdornment,
  InputLabel,
  MenuItem,
  OutlinedInput,
  Paper,
  Select,
  Stack,
  Switch,
  IconButton,
  TextField,
  Typography
} from '@mui/material';

import { showError, showSuccess, trims } from 'utils/common';
import { API } from 'utils/api';
import { createFilterOptions } from '@mui/material/Autocomplete';
import { priceType, ValueFormatter } from './util';
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import { Icon } from '@iconify/react';
import { useTranslation } from 'react-i18next';
import ToggleButtonGroup from 'ui-component/ToggleButton';
import Decimal from 'decimal.js';
import { ExtraRatiosSelector } from './ExtraRatiosSelector';

const icon = <CheckBoxOutlineBlankIcon fontSize="small" />;
const checkedIcon = <CheckBoxIcon fontSize="small" />;

const filter = createFilterOptions();

// 分时倍率：界面上按「元/百万 tokens」填写，保存时换算成倍率（1 倍率 = 0.014 元/千 tokens）
const RMB_PER_RATE = 14;

const emptyTimePeriod = () => ({
  name: '',
  start: '22:00',
  end: '08:00',
  input: '',
  output: '',
  cached: ''
});

// 接口数据 -> 表单（倍率转成元/百万）
const timePricesToForm = (timePrices) => {
  if (!timePrices || !timePrices.enabled || !Array.isArray(timePrices.periods) || timePrices.periods.length === 0) {
    return { enabled: false, periods: [emptyTimePeriod()] };
  }
  return {
    enabled: true,
    periods: timePrices.periods.map((period) => ({
      name: period.name || '',
      start: period.start || '',
      end: period.end || '',
      input: period.input ? Number((period.input * RMB_PER_RATE).toFixed(6)) : '',
      output: period.output ? Number((period.output * RMB_PER_RATE).toFixed(6)) : '',
      cached: period.cached === undefined || period.cached === null ? '' : String(period.cached)
    }))
  };
};

// 表单 -> 接口数据（元/百万 转回倍率）
const timePricesToApi = (timePrices) => {
  if (!timePrices || !timePrices.enabled) {
    return null;
  }
  const periods = (timePrices.periods || [])
    .filter((period) => period.start && period.end)
    .map((period) => {
      const item = {
        name: period.name || `${period.start}-${period.end}`,
        start: period.start,
        end: period.end,
        input: Number(period.input) > 0 ? Number((Number(period.input) / RMB_PER_RATE).toFixed(6)) : 0,
        output: Number(period.output) > 0 ? Number((Number(period.output) / RMB_PER_RATE).toFixed(6)) : 0
      };
      if (period.cached !== '' && period.cached !== undefined && period.cached !== null && Number(period.cached) > 0) {
        item.cached = Number(period.cached);
      }
      return item;
    });
  if (periods.length === 0) {
    return null;
  }
  return { enabled: true, periods };
};

// 单一模式下的表单验证
const validateSingleMode = (t, values, rows) => {
  if (values.model === '') {
    return t('pricing_edit.requiredModelName');
  }

  if (values.type !== 'tokens' && values.type !== 'times') {
    return t('pricing_edit.typeCheck');
  }

  if (values.channel_type <= 0) {
    return t('pricing_edit.channelTypeErr2');
  }

  // 判断model是否是唯一值
  if (rows && rows.filter((r) => r.model === values.model && (values.isNew || r.id !== values.id)).length > 0) {
    return t('pricing_edit.modelNameRe');
  }

  // 按次计费时只验证单次价格（input字段）
  if (values.type === 'times') {
    if (values.input === '' || values.input < 0) {
      return t('pricing_edit.inputVal');
    }
  } else {
    // 按Token计费时验证输入输出价格
    if (values.input === '' || values.input < 0) {
      return t('pricing_edit.inputVal');
    }
    if (values.output === '' || values.output < 0) {
      return t('pricing_edit.outputVal');
    }
  }
  // 分时倍率校验
  if (values.time_prices?.enabled) {
    const periods = (values.time_prices.periods || []).filter((period) => period.start && period.end);
    if (periods.length === 0) {
      return t('pricing_edit.timePricesEmpty');
    }
    for (const period of periods) {
      if (Number(period.input) <= 0 && (values.type === 'times' || Number(period.output) <= 0)) {
        return t('pricing_edit.timePricesZero');
      }
    }
  }

  return false;
};

// 多选模式下的表单验证
const getValidationSchema = (t) =>
  Yup.object().shape({
    is_edit: Yup.boolean(),
    type: Yup.string().oneOf(['tokens', 'times'], t('pricing_edit.typeErr')).required(t('pricing_edit.requiredType')),
    channel_type: Yup.number().min(1, t('pricing_edit.channelTypeErr')).required(t('pricing_edit.requiredChannelType')),
    input: Yup.number()
      .required(t('pricing_edit.requiredInput'))
      .test('isPositive', t('pricing_edit.inputVal'), (value) => value !== '' && value >= 0),
    output: Yup.number().when('type', {
      is: 'tokens',
      then: (schema) =>
        schema
          .required(t('pricing_edit.requiredOutput'))
          .test('isPositive', t('pricing_edit.outputVal'), (value) => value !== '' && value >= 0),
      otherwise: (schema) =>
        schema
          .notRequired()
          .nullable()
          .transform((value) => (value === undefined || value === '' ? null : value))
    }),
    models: Yup.array().min(1, t('pricing_edit.requiredModels'))
  });

// 多选模式初始值
const multipleOriginInputs = {
  is_edit: false,
  type: 'tokens',
  channel_type: 1,
  input: 0,
  output: 0,
  locked: false,
  models: [],
  extra_ratios: {}
};

// 单一模式初始值
const singleOriginInputs = {
  model: '',
  type: 'tokens',
  channel_type: 1,
  input: 0,
  output: 0,
  locked: false,
  extra_ratios: {},
  time_prices: { enabled: false, periods: [emptyTimePeriod()] }
};

const EditModal = ({
  open,
  pricesItem,
  onCancel,
  onOk,
  ownedby,
  noPriceModel,
  singleMode = false,
  price = null,
  rows = [],
  onSaveSingle = null,
  unit = 'K'
}) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const [inputs, setInputs] = useState(singleMode ? singleOriginInputs : multipleOriginInputs);
  const [selectModel, setSelectModel] = useState([]);
  const [errors, setErrors] = useState({});

  const [unitType, setUnitType] = useState('rate');
  const [localUnit, setLocalUnit] = useState(unit);

  // 当外部unit变化时同步本地unit
  useEffect(() => {
    setLocalUnit(unit);
  }, [unit]);

  const calculateRate = useCallback(
    (price) => {
      if (price === '') {
        return 0;
      }
      if (unitType === 'rate') {
        return Number(price);
      }

      let priceValue = new Decimal(price);

      if (localUnit === 'M') {
        priceValue = priceValue.div(1000);
      }

      switch (unitType) {
        case 'USD':
          priceValue = priceValue.div(0.002);
          break;
        case 'RMB':
          priceValue = priceValue.div(0.014);
          break;
      }

      return Number(priceValue.toFixed(4));
    },
    [unitType, localUnit]
  );

  const unitTypeOptions = [
    { value: 'rate', label: t('modelpricePage.rate') },
    { value: 'USD', label: 'USD' },
    { value: 'RMB', label: 'RMB' }
  ];

  const lockedOptions = [
    { value: true, label: t('pricing_edit.locked') },
    { value: false, label: t('pricing_edit.unlocked') }
  ];

  const unitOptions = [
    { value: 'K', label: 'K' },
    { value: 'M', label: 'M' }
  ];

  const handleEndAdornment = useCallback(
    (value) => {
      let endAdornment = '';

      switch (unitType) {
        case 'rate':
          endAdornment = ValueFormatter(value);
          break;
        case 'USD':
        case 'RMB':
          endAdornment = Number(value) === 0 ? 'Free' : calculateRate(value) + ' Rate';
          break;
      }

      return endAdornment;
    },
    [unitType, calculateRate]
  );

  const handleStartAdornment = useCallback(() => {
    switch (unitType) {
      case 'rate':
        return 'Rate：';
      case 'USD':
        return `USD(${localUnit})：`;
      case 'RMB':
        return `RMB(${localUnit})：`;
    }
  }, [unitType, localUnit]);

  // 表单提交处理
  const submit = async (values, { setErrors, setStatus, setSubmitting }) => {
    setSubmitting(true);

    // Ensure extra_ratios values are numbers
    if (values.extra_ratios) {
      const processedRatios = {};
      Object.keys(values.extra_ratios).forEach((key) => {
        processedRatios[key] = Number(values.extra_ratios[key]);
      });
      values.extra_ratios = processedRatios;
    }

    // 单一模式处理
    if (singleMode) {
      // 验证表单
      const validationError = validateSingleMode(t, values, rows);
      if (validationError) {
        setStatus({ success: false });
        setErrors({ submit: validationError });
        return;
      }

      try {
        if (onSaveSingle) {
          const calculatedInput = calculateRate(values.input);
          const calculatedOutput = values.type === 'times' ? calculatedInput : calculateRate(values.output);
          await onSaveSingle({
            ...values,
            input: calculatedInput,
            output: calculatedOutput,
            time_prices: timePricesToApi(values.time_prices)
          });
        }
        setSubmitting(false);
        return;
      } catch (error) {
        setStatus({ success: false });
        setErrors({ submit: error.message });
        return;
      }
    }

    // 多选模式处理
    values.models = trims(values.models);
    try {
      const calculatedInput = calculateRate(values.input);
      const calculatedOutput = values.type === 'times' ? calculatedInput : calculateRate(values.output);
      const res = await API.post(`/api/prices/multiple`, {
        original_models: inputs.models,
        models: values.models,
        price: {
          model: 'batch',
          type: values.type,
          channel_type: values.channel_type,
          input: calculatedInput,
          output: calculatedOutput,
          locked: values.locked,
          extra_ratios: values.extra_ratios
        }
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('common.saveSuccess'));
        setSubmitting(false);
        setStatus({ success: true });
        onOk(true);
        return;
      } else {
        setStatus({ success: false });
        showError(message);
        setErrors({ submit: message });
      }
    } catch (error) {
      setStatus({ success: false });
      showError(error.message);
      setErrors({ submit: error.message });
      return;
    }
    onOk();
  };

  // 处理表单字段变化 (单模式用)
  const handleChange = (event) => {
    if (!singleMode) return; // 单一模式专用

    const { name, value, checked } = event.target;

    let finalValue;
    if (name === 'input' || name === 'output') {
      finalValue = value;
    } else {
      finalValue = name === 'locked' ? checked : value;
    }

    setInputs((prev) => ({
      ...prev,
      [name]: finalValue
    }));

    if (errors[name]) {
      setErrors((prev) => ({ ...prev, [name]: null }));
    }
  };

  // 处理extra_ratios变化 (单模式用)
  const handleChangeExtraRatios = (newExtraRatios) => {
    if (!singleMode) return; // 单一模式专用

    setInputs((prev) => ({
      ...prev,
      extra_ratios: newExtraRatios
    }));
  };

  // 分时倍率开关
  const handleToggleTimePrices = (enabled) => {
    setInputs((prev) => {
      const current = prev.time_prices || { enabled: false, periods: [] };
      const periods = current.periods && current.periods.length > 0 ? current.periods : [emptyTimePeriod()];
      return { ...prev, time_prices: { enabled, periods } };
    });
  };

  // 修改某个时段的字段
  const handleTimePeriodChange = (index, field, value) => {
    setInputs((prev) => {
      const current = prev.time_prices || { enabled: true, periods: [] };
      const periods = (current.periods || []).map((period, i) => (i === index ? { ...period, [field]: value } : period));
      return { ...prev, time_prices: { enabled: true, periods } };
    });
  };

  const handleAddTimePeriod = () => {
    setInputs((prev) => {
      const current = prev.time_prices || { enabled: true, periods: [] };
      return { ...prev, time_prices: { enabled: true, periods: [...(current.periods || []), emptyTimePeriod()] } };
    });
  };

  const handleRemoveTimePeriod = (index) => {
    setInputs((prev) => {
      const current = prev.time_prices || { enabled: true, periods: [] };
      const periods = (current.periods || []).filter((period, i) => i !== index);
      return { ...prev, time_prices: { enabled: true, periods: periods.length > 0 ? periods : [emptyTimePeriod()] } };
    });
  };

  useEffect(() => {
    if (singleMode) {
      // 单一模式初始化表单
      if (price) {
        setInputs({
          ...price,
          extra_ratios: price.extra_ratios || {},
          time_prices: timePricesToForm(price.time_prices)
        });
      } else {
        setInputs({ ...singleOriginInputs, time_prices: timePricesToForm(null) });
      }
      setErrors({});
    } else {
      // 多选模式初始化
      if (pricesItem) {
        setSelectModel(pricesItem.models.concat(noPriceModel));
        setInputs(pricesItem);
      } else {
        setSelectModel(noPriceModel);
        setInputs(multipleOriginInputs);
      }
    }
  }, [singleMode, price, pricesItem, noPriceModel]);

  useEffect(() => {
    if (open) {
      setUnitType('rate');
      setLocalUnit('K');
    }
  }, [open]);

  // 渲染类型选择表单
  const renderTypeSelector = (formProps) => {
    const { errors = {}, touched = {}, handleBlur, handleChange: formikHandleChange, values = {} } = formProps || {};

    return (
      <FormControl
        fullWidth
        error={singleMode ? !!errors.type : Boolean(touched?.type && errors?.type)}
        sx={{ ...theme.typography.otherInput }}
      >
        <InputLabel htmlFor="type-label">{t('pricing_edit.type')}</InputLabel>
        <Select
          id="type-label"
          label={t('pricing_edit.type')}
          value={singleMode ? inputs.type : values?.type}
          name="type"
          onBlur={handleBlur}
          onChange={singleMode ? handleChange : formikHandleChange}
          MenuProps={{
            PaperProps: {
              style: {
                maxHeight: 200
              }
            }
          }}
        >
          {Object.values(priceType).map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </Select>
        {!singleMode && touched?.type && errors?.type && (
          <FormHelperText error id="helper-tex-type-label">
            {errors.type}
          </FormHelperText>
        )}
      </FormControl>
    );
  };

  // 渲染渠道类型选择表单
  const renderChannelTypeSelector = (formProps) => {
    const { errors = {}, touched = {}, handleBlur, handleChange: formikHandleChange, values = {} } = formProps || {};

    return (
      <FormControl
        fullWidth
        error={singleMode ? !!errors.channel_type : Boolean(touched?.channel_type && errors?.channel_type)}
        sx={{ ...theme.typography.otherInput }}
      >
        <InputLabel htmlFor="channel_type-label">{t('pricing_edit.channelType')}</InputLabel>
        <Select
          id="channel_type-label"
          label={t('pricing_edit.channelType')}
          value={singleMode ? inputs.channel_type : values?.channel_type}
          name="channel_type"
          onBlur={handleBlur}
          onChange={singleMode ? handleChange : formikHandleChange}
          MenuProps={{
            PaperProps: {
              style: {
                maxHeight: 200
              }
            }
          }}
        >
          {ownedby.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </Select>
        {!singleMode && touched?.channel_type && errors?.channel_type && (
          <FormHelperText error id="helper-tex-channel_type-label">
            {errors.channel_type}
          </FormHelperText>
        )}
      </FormControl>
    );
  };

  // 渲染单位类型切换按钮组
  const renderUnitTypeToggle = () => (
    <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
      <Stack direction="row" spacing={2}>
        <ToggleButtonGroup
          value={unitType}
          onChange={(event, newUnitType) => {
            setUnitType(newUnitType);
          }}
          options={unitTypeOptions}
          aria-label="unit toggle"
        />

        <ToggleButtonGroup
          value={localUnit}
          onChange={(event, newUnit) => {
            setLocalUnit(newUnit);
          }}
          options={unitOptions}
          aria-label="unit toggle"
        />
      </Stack>
    </FormControl>
  );

  // 渲染输入价格表单
  const renderInputField = (formProps) => {
    const { errors = {}, touched = {}, handleBlur, handleChange: formikHandleChange, values = {} } = formProps || {};

    // 根据模式选择正确的处理函数和值
    const value = singleMode ? inputs.input : values.input;
    const onChange = singleMode ? handleChange : formikHandleChange;
    const errorState = singleMode ? !!errors.input : Boolean(touched?.input && errors?.input);

    // 根据类型决定 label
    const currentType = singleMode ? inputs.type : values?.type;
    const inputLabel = currentType === 'times' ? t('modelpricePage.timesPrice') : t('modelpricePage.inputMultiplier');

    return (
      <FormControl fullWidth error={errorState} sx={{ ...theme.typography.otherInput }}>
        <InputLabel htmlFor="channel-input-label">{inputLabel}</InputLabel>
        <OutlinedInput
          id="channel-input-label"
          label={inputLabel}
          type="number"
          value={value}
          name="input"
          startAdornment={<InputAdornment position="start">{handleStartAdornment()}</InputAdornment>}
          endAdornment={<InputAdornment position="end">{handleEndAdornment(value)}</InputAdornment>}
          onBlur={handleBlur}
          onChange={onChange}
          aria-describedby="helper-text-channel-input-label"
        />

        {(singleMode && errors.input) || (!singleMode && touched?.input && errors?.input) ? (
          <FormHelperText error id="helper-tex-channel-input-label">
            {errors.input}
          </FormHelperText>
        ) : null}
      </FormControl>
    );
  };

  // 渲染输出价格表单
  const renderOutputField = (formProps) => {
    const { errors = {}, touched = {}, handleBlur, handleChange: formikHandleChange, values = {} } = formProps || {};

    // 根据模式选择正确的处理函数和值
    const value = singleMode ? inputs.output : values.output;
    const onChange = singleMode ? handleChange : formikHandleChange;
    const errorState = singleMode ? !!errors.output : Boolean(touched?.output && errors?.output);

    return (
      <FormControl fullWidth error={errorState} sx={{ ...theme.typography.otherInput }}>
        <InputLabel htmlFor="channel-output-label">{t('modelpricePage.outputMultiplier')}</InputLabel>
        <OutlinedInput
          id="channel-output-label"
          label={t('modelpricePage.outputMultiplier')}
          type="number"
          value={value}
          name="output"
          startAdornment={<InputAdornment position="start">{handleStartAdornment()}</InputAdornment>}
          endAdornment={<InputAdornment position="end">{handleEndAdornment(value)}</InputAdornment>}
          onBlur={handleBlur}
          onChange={onChange}
          aria-describedby="helper-text-channel-output-label"
        />

        {(singleMode && errors.output) || (!singleMode && touched?.output && errors?.output) ? (
          <FormHelperText error id="helper-tex-channel-output-label">
            {errors.output}
          </FormHelperText>
        ) : null}
      </FormControl>
    );
  };

  // 渲染锁定切换按钮组
  const renderLockedToggle = (formProps) => {
    const { handleChange: formikHandleChange, values = {} } = formProps || {};

    // 在单模式下，我们需要使用不同的处理方式
    const handleLockChange = (event, newLocked) => {
      if (singleMode) {
        // 在单模式下，直接更新inputs状态
        handleChange({
          target: {
            name: 'locked',
            checked: newLocked
          }
        });
      } else {
        // 在多模式下，使用Formik的handleChange
        formikHandleChange({
          target: {
            name: 'locked',
            value: newLocked
          }
        });
      }
    };

    return (
      <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
        <Stack direction="row" spacing={2}>
          <ToggleButtonGroup
            value={singleMode ? inputs.locked : values?.locked}
            onChange={handleLockChange}
            options={lockedOptions}
            aria-label="locked toggle"
          />
        </Stack>
      </FormControl>
    );
  };

  // 渲染额外比率选择器
  const renderExtraRatioSelector = (formProps) => {
    const { setFieldValue, values = {} } = formProps || {};

    if (singleMode) {
      return (
        <Paper variant="outlined" sx={{ p: 2, mt: 2 }}>
          <ExtraRatiosSelector value={inputs.extra_ratios} onChange={handleChangeExtraRatios} />
        </Paper>
      );
    }

    return (
      <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
        <ExtraRatiosSelector
          value={values.extra_ratios || {}}
          onChange={(newExtraRatios) => {
            setFieldValue('extra_ratios', newExtraRatios);
          }}
          handleStartAdornment={handleStartAdornment}
        />
      </FormControl>
    );
  };

  // 渲染分时倍率配置（单模式）
  const renderTimePrices = () => {
    const timePrices = inputs.time_prices || { enabled: false, periods: [] };
    const periods = timePrices.periods && timePrices.periods.length > 0 ? timePrices.periods : [emptyTimePeriod()];

    return (
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Stack direction="row" alignItems="center" justifyContent="space-between">
          <Box>
            <Typography variant="subtitle2">{t('pricing_edit.timePrices')}</Typography>
            <Typography variant="caption" color="text.secondary">
              {t('pricing_edit.timePricesTip')}
            </Typography>
          </Box>
          <Switch checked={!!timePrices.enabled} onChange={(event) => handleToggleTimePrices(event.target.checked)} />
        </Stack>

        {timePrices.enabled && (
          <Stack spacing={1.5} sx={{ mt: 2 }}>
            {periods.map((period, index) => (
              <Stack key={index} direction={{ xs: 'column', sm: 'row' }} spacing={1} alignItems={{ xs: 'stretch', sm: 'center' }}>
                <TextField
                  size="small"
                  label={t('pricing_edit.periodName')}
                  value={period.name}
                  onChange={(event) => handleTimePeriodChange(index, 'name', event.target.value)}
                  sx={{ width: { xs: '100%', sm: 110 } }}
                />
                <TextField
                  size="small"
                  type="time"
                  label={t('pricing_edit.periodStart')}
                  value={period.start}
                  onChange={(event) => handleTimePeriodChange(index, 'start', event.target.value)}
                  InputLabelProps={{ shrink: true }}
                  sx={{ width: { xs: '100%', sm: 120 } }}
                />
                <TextField
                  size="small"
                  type="time"
                  label={t('pricing_edit.periodEnd')}
                  value={period.end}
                  onChange={(event) => handleTimePeriodChange(index, 'end', event.target.value)}
                  InputLabelProps={{ shrink: true }}
                  sx={{ width: { xs: '100%', sm: 120 } }}
                />
                <TextField
                  size="small"
                  type="number"
                  label={t('pricing_edit.periodInput')}
                  value={period.input}
                  onChange={(event) => handleTimePeriodChange(index, 'input', event.target.value)}
                  InputProps={{ endAdornment: <InputAdornment position="end">¥/M</InputAdornment> }}
                  sx={{ width: { xs: '100%', sm: 130 } }}
                />
                <TextField
                  size="small"
                  type="number"
                  label={t('pricing_edit.periodOutput')}
                  value={period.output}
                  disabled={inputs.type === 'times'}
                  onChange={(event) => handleTimePeriodChange(index, 'output', event.target.value)}
                  InputProps={{ endAdornment: <InputAdornment position="end">¥/M</InputAdornment> }}
                  sx={{ width: { xs: '100%', sm: 130 } }}
                />
                <TextField
                  size="small"
                  type="number"
                  label={t('pricing_edit.periodCached')}
                  value={period.cached}
                  onChange={(event) => handleTimePeriodChange(index, 'cached', event.target.value)}
                  sx={{ width: { xs: '100%', sm: 120 } }}
                />
                <IconButton size="small" onClick={() => handleRemoveTimePeriod(index)} title={t('pricing_edit.removePeriod')}>
                  <Icon icon="mdi:close" fontSize={16} />
                </IconButton>
              </Stack>
            ))}
            <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
              <Button size="small" variant="outlined" onClick={handleAddTimePeriod}>
                {t('pricing_edit.addPeriod')}
              </Button>
              <Typography variant="caption" color="text.secondary">
                {t('pricing_edit.timePricesHint')}
              </Typography>
            </Stack>
          </Stack>
        )}
      </Paper>
    );
  };

  // 渲染模型选择器 (多模式特有)
  const renderModelSelector = (formProps) => {
    if (!formProps) return null;

    const { errors = {}, handleBlur, handleChange, values = {} } = formProps;

    return (
      <FormControl fullWidth sx={{ ...theme.typography.otherInput }}>
        <Autocomplete
          multiple
          freeSolo
          id="channel-models-label"
          options={selectModel || []}
          value={values.models || []}
          onChange={(e, value) => {
            const event = {
              target: {
                name: 'models',
                value: value
              }
            };
            handleChange(event);
          }}
          onBlur={handleBlur}
          disableCloseOnSelect
          getOptionLabel={(option) => option || ''}
          renderInput={(params) => <TextField {...params} name="models" error={Boolean(errors.models)} label={t('pricing_edit.model')} />}
          filterOptions={(options, params) => {
            const filtered = filter(options, params);
            const { inputValue } = params;
            const isExisting = options.some((option) => inputValue === option);
            if (inputValue !== '' && !isExisting) {
              filtered.push(inputValue);
            }
            return filtered;
          }}
          renderOption={(props, option, { selected }) => (
            <li {...props}>
              <Checkbox icon={icon} checkedIcon={checkedIcon} style={{ marginRight: 8 }} checked={selected} />
              {option}
            </li>
          )}
        />
        {errors.models ? (
          <FormHelperText error id="helper-tex-channel-models-label">
            {errors.models}
          </FormHelperText>
        ) : (
          <FormHelperText id="helper-tex-channel-models-label"> {t('pricing_edit.modelTip')} </FormHelperText>
        )}
      </FormControl>
    );
  };

  // 渲染操作按钮
  const renderActions = (formProps) => {
    const { isSubmitting = false } = formProps || {};

    return (
      <DialogActions>
        <Button onClick={onCancel}>{t('common.cancel')}</Button>
        {singleMode ? (
          <Button
            onClick={() =>
              submit(inputs, {
                setErrors,
                setStatus: () => {},
                setSubmitting: () => {}
              })
            }
            variant="contained"
            color="primary"
          >
            {t('common.submit')}
          </Button>
        ) : (
          <Button disableElevation disabled={isSubmitting} type="submit" variant="contained" color="primary">
            {t('common.submit')}
          </Button>
        )}
      </DialogActions>
    );
  };

  return (
    <Dialog open={open} onClose={onCancel} fullWidth maxWidth="md">
      <DialogTitle sx={{ margin: '0px', fontWeight: 700, lineHeight: '1.55556', padding: '24px', fontSize: '1.125rem' }}>
        {singleMode ? (price ? t('common.edit') : t('common.create')) : pricesItem ? t('common.edit') : t('common.create')}
      </DialogTitle>
      <Divider />
      <DialogContent>
        {singleMode ? (
          // 单一模式表单
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label={t('pricing_edit.name')}
              name="model"
              value={inputs.model}
              onChange={handleChange}
              fullWidth
              error={!!errors.model}
              helperText={errors.model}
            />

            {renderTypeSelector()}
            {renderChannelTypeSelector()}
            {renderUnitTypeToggle()}

            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              {renderInputField()}
              {inputs.type === 'tokens' && renderOutputField()}
            </Stack>

            {renderLockedToggle()}
            <Alert severity="warning">{t('pricing_edit.lockedTip')}</Alert>

            {renderExtraRatioSelector()}
            {renderTimePrices()}

            {errors.general && (
              <Typography color="error" variant="body2">
                {errors.general}
              </Typography>
            )}

            {renderActions()}
          </Stack>
        ) : (
          // 多选模式表单
          <Formik initialValues={inputs} enableReinitialize validationSchema={getValidationSchema(t)} onSubmit={submit}>
            {(formProps) => (
              <form noValidate onSubmit={formProps.handleSubmit}>
                {renderTypeSelector(formProps)}
                {renderChannelTypeSelector(formProps)}
                {renderUnitTypeToggle()}
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  {renderInputField(formProps)}
                  {formProps.values.type === 'tokens' && renderOutputField(formProps)}
                </Stack>
                {renderModelSelector(formProps)}
                {renderLockedToggle(formProps)}
                <Alert severity="warning">{t('pricing_edit.lockedTip')}</Alert>
                {renderExtraRatioSelector(formProps)}
                {renderActions(formProps)}
              </form>
            )}
          </Formik>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default EditModal;

EditModal.propTypes = {
  open: PropTypes.bool,
  pricesItem: PropTypes.oneOfType([PropTypes.object, PropTypes.any]),
  onCancel: PropTypes.func,
  onOk: PropTypes.func,
  ownedby: PropTypes.array,
  noPriceModel: PropTypes.array,
  // 以下是单一模式专用
  singleMode: PropTypes.bool,
  price: PropTypes.object,
  rows: PropTypes.array,
  onSaveSingle: PropTypes.func,
  unit: PropTypes.string
};
