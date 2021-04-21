import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import {
  Button, Form, Row, Col,
} from 'react-bootstrap';

import AppContext from '../../../AppContext';
import TasksAPI from '../../../api/TasksAPI';
import FormGroup from '../../_shared/FormGroup/FormGroup';
import EditSubtasks from './EditSubtasks/EditSubtasks';
import inputType from '../../../misc/inputType';
import ValidateTask from '../../../validation/ValidateTask';

import logo from './edittask.svg';
import './edittask.sass';

const EditTask = ({
  id, title, description, subtasks, toggleOff,
}) => {
  const { activeBoard, loadBoard, notify } = useContext(AppContext);
  const [newTitle, setNewTitle] = useState(title);
  const [newDescription, setNewDescription] = useState(description);
  const [newSubtasks, setNewSubtasks] = useState({
    value: '',
    list: subtasks,
  });
  const [titleError, setTitleError] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();

    const clientTitleError = ValidateTask.title(newTitle);

    if (clientTitleError) {
      setTitleError(clientTitleError);
    } else {
      TasksAPI
        .patch(id, {
          title: newTitle,
          description: newDescription,
          column: activeBoard.columns[0].id,
          subtasks: newSubtasks.list,
        })
        .then(() => {
          loadBoard();
          toggleOff();
        })
        .catch((err) => {
          const serverTitleError = err?.response?.data?.title || '';

          if (serverTitleError) {
            setTitleError(serverTitleError);
          } else {
            notify(
              'Unable to edit task.',
              `${err?.message || 'Server Error'}.`,
            );
          }
        });
    }
  };

  return (
    <div className="EditTask">
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
          value={newTitle}
          setValue={setNewTitle}
          error={titleError}
        />

        <FormGroup
          type={inputType.TEXTAREA}
          label="description"
          value={newDescription}
          setValue={setNewDescription}
        />

        <EditSubtasks
          subtasks={newSubtasks}
          setSubtasks={setNewSubtasks}
        />

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
              className="Button GoButton"
              type="submit"
              aria-label="submit"
            >
              SUBMIT
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

EditTask.propTypes = {
  id: PropTypes.number.isRequired,
  title: PropTypes.string.isRequired,
  description: PropTypes.string.isRequired,
  subtasks: PropTypes.arrayOf({
    id: PropTypes.number.isRequired,
    title: PropTypes.string.isRequired,
    order: PropTypes.number.isRequired,
    done: PropTypes.bool.isRequired,
  }).isRequired,
  toggleOff: PropTypes.func.isRequired,
};

export default EditTask;
