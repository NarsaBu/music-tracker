import type { Preview } from '@storybook/react-vite';
import '@/shared/ui/theme/global.css';

const preview: Preview = {
  globalTypes: {
    theme: {
      description: 'Тема стойки',
      toolbar: {
        title: 'Theme',
        items: [
          { value: 'day', title: 'Day' },
          { value: 'night', title: 'Night (lamp)' },
        ],
      },
    },
  },
  initialGlobals: { theme: 'day' },
  decorators: [
    (Story, context) => {
      document.documentElement.dataset.theme = context.globals.theme as string;
      return (
        <div style={{ padding: 24 }}>
          <Story />
        </div>
      );
    },
  ],
};

export default preview;
