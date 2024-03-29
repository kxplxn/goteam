const columnOrder = {
  INBOX: 'inbox',
  READY: 'ready',
  GO: 'go!',
  DONE: 'done',
  parseInt: (order) => {
    switch (order) {
      case 0:
        return columnOrder.INBOX;
      case 1:
        return columnOrder.READY;
      case 2:
        return columnOrder.GO;
      case 3:
        return columnOrder.DONE;
      default:
        return Error('column order must be between 0 and 3');
    }
  },
};

export default columnOrder;
