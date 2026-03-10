/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import {
  Button,
  InputNumber,
  Modal,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

const TokenRPMModal = ({
  visible,
  token,
  explicitRPM,
  effectiveRPM,
  defaultTokenRPM,
  onClose,
  refreshTokenRPMOverview,
  t,
}) => {
  const [loading, setLoading] = useState(false);
  const [rpm, setRPM] = useState(0);

  useEffect(() => {
    setRPM(explicitRPM || 0);
  }, [explicitRPM, token?.id, visible]);

  const onSubmit = async () => {
    const nextRPM = Number(rpm || 0);
    if (!Number.isInteger(nextRPM) || nextRPM < 0) {
      showError(t('RPM 必须是大于等于 0 的整数'));
      return;
    }
    setLoading(true);
    try {
      const res =
        nextRPM > 0
          ? await API.put('/api/token/rpm', {
              token_id: token.id,
              rpm: nextRPM,
            })
          : await API.delete(`/api/token/rpm?token_id=${token.id}`);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      await refreshTokenRPMOverview();
      showSuccess(
        nextRPM > 0 ? t('令牌 RPM 已保存') : t('令牌 RPM 已恢复为默认值'),
      );
      onClose();
    } catch (error) {
      showError(error.message || t('令牌 RPM 保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t('设置令牌 RPM')}
      visible={visible}
      onCancel={onClose}
      closeOnEsc
      footer={
        <Space>
          <Button onClick={onClose}>{t('取消')}</Button>
          <Button type='primary' loading={loading} onClick={onSubmit}>
            {t('保存')}
          </Button>
        </Space>
      }
    >
      <div className='flex flex-col gap-3'>
        <Text strong>{token?.name || '-'}</Text>
        <InputNumber
          min={0}
          step={1}
          value={rpm}
          onChange={(value) => setRPM(value || 0)}
          style={{ width: '100%' }}
          placeholder={t('0 表示继承默认 RPM')}
        />
        <Text type='secondary'>
          {t('当前生效值')}：{effectiveRPM || 0} RPM
        </Text>
        <Text type='secondary'>
          {t('当前默认值')}：{defaultTokenRPM || 0} RPM
        </Text>
        <Text type='secondary'>
          {t('当前单独设置值')}：{explicitRPM || 0} RPM
        </Text>
        <Text type='secondary'>
          {t('填写 0 表示删除单独设置，回退到默认 RPM')}
        </Text>
      </div>
    </Modal>
  );
};

export default TokenRPMModal;
