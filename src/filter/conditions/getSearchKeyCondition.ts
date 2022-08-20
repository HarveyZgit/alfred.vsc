import { Condition } from '../utils';

const getSearchKeyCondition: Condition.Checker = ({ searchKeyReg }) => {
  return (record) => searchKeyReg.test(record.name);
};

export default getSearchKeyCondition;
