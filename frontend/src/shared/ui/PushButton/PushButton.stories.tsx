import type { Meta, StoryObj } from '@storybook/react-vite';
import { PushButton } from './PushButton';

const meta = { title: 'UI/PushButton', component: PushButton } satisfies Meta<typeof PushButton>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = { args: { variant: 'primary', children: 'Сохранить' } };
export const PrimaryWithLed: Story = {
  args: { variant: 'primary', led: true, children: 'Сохранить' },
};
export const Secondary: Story = { args: { variant: 'secondary', children: 'Отмена' } };
export const Disabled: Story = {
  args: { variant: 'primary', disabled: true, children: 'Сохранить' },
};
