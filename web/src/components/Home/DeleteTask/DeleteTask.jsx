import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import {
  Button, Form, Row, Col,
} from 'react-bootstrap';

import AppContext from '../../../AppContext';
import TaskAPI from '../../../api/TaskAPI';
import FormGroup from '../../_shared/FormGroup/FormGroup';
import EditSubtasks from '../EditTask/EditSubtasks/EditSubtasks';
import inputType from '../../../misc/inputType';

import logo from './deletetask.svg';
import './deletetask.sass';

const DeleteTask = ({
  id, title, description, subtasks, colNo, toggleOff,
}) => {
  const { activeBoard, setActiveBoard, notify } = useContext(AppContext);

  const handleSubmit = (e) => {
    e.preventDefault();

    // Keep an initial state to avoid loadBoard() on API error
    const initialActiveBoard = activeBoard;

    // Update client state to avoid load time
    setActiveBoard({
      ...activeBoard,
      columns: activeBoard.columns.map((column, i) => (
        i === colNo ? {
          ...column,
          tasks: column.tasks.filter((task) => (task.id !== id)),
        } : column
      )),
    });

    // Delete task in database
    TaskAPI
      .delete(id)
      .then(toggleOff)
      .catch((err) => {
        notify(
          'Unable to delete task.',
          `${err?.response?.data?.error || 'Server Error'}.`,
        );
        setActiveBoard(initialActiveBoard);
      });
  };

  return (
    <div className="DeleteTask">
      <Form
        className="Form"
        onSubmit={handleSubmit}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="HeaderWrapper">
          <img className="Header" alt="logo" src={logo} />
        </div>

        <FormGroup
          type={inputType.TEXT}
          label="title"
          value={title}
          disabled
        />

        <FormGroup
          type={inputType.TEXTAREA}
          label="description"
          value={description}
          disabled
        />

        {subtasks.length > 0
          && <EditSubtasks subtasks={{ list: subtasks }} />}

        <Row className="ButtonWrapper">
          <Col className="ButtonCol">
            <Button
              className="Button CancelButton"
              type="button"
              aria-label="cancel"
              onClick={toggleOff}
            >
              CANCEL
            </Button>
          </Col>
          <Col className="ButtonCol">
            <Button
              className="Button DeleteButton"
              type="submit"
              aria-label="submit"
            >
              DELETE
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

DeleteTask.propTypes = {
  id: PropTypes.string.isRequired,
  title: PropTypes.string.isRequired,
  description: PropTypes.string,
  subtasks: PropTypes.arrayOf(
    PropTypes.exact({
      title: PropTypes.string.isRequired,
      done: PropTypes.bool.isRequired,
    }),
  ).isRequired,
  colNo: PropTypes.number.isRequired,
  toggleOff: PropTypes.func.isRequired,
};

DeleteTask.defaultProps = {
  description: null,
};

export default DeleteTask;
